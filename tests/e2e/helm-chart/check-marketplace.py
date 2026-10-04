#!/usr/bin/env python3
"""Render-only activation checks. Requires Helm and PyYAML; never contacts AWS/Kubernetes."""

import copy
import json
from pathlib import Path
import subprocess
import tempfile

import yaml


ROOT = Path(__file__).resolve().parents[3]
COMPONENTS = ["autoscaler", "scheduler", "enterprise-instrumentor", "enterprise-odiglet",
              "enterprise-collector", "enterprise-ui", "enterprise-agents", "cli"]
REGISTRY = "709825985650.dkr.ecr.us-east-1.amazonaws.com/odigos"
ROLE = "arn:aws:iam::123456789012:role/test-marketplace-license"
VALUES = {
    "onPremToken": "render-test-token",
    "marketplace": {"enabled": True,
                    "serviceAccountAnnotations": {"eks.amazonaws.com/role-arn": ROLE}},
    "images": {c: f"{REGISTRY}/odigos-{c}:test" for c in COMPONENTS},
}


def render(values, expected_error=None):
    with tempfile.TemporaryDirectory() as directory:
        path = Path(directory) / "values.json"
        path.write_text(json.dumps(values))
        result = subprocess.run(["helm", "template", "odigos", str(ROOT / "helm/odigos"),
                                 "--namespace", "test-marketplace", "--kube-version", "1.34.0",
                                 "--values", str(path)], capture_output=True, text=True, check=False)
    if expected_error:
        assert result.returncode != 0 and expected_error in result.stderr, result.stderr
        return []
    assert result.returncode == 0, result.stderr
    return [d for d in yaml.safe_load_all(result.stdout) if d]


def workloads(documents):
    return {d["metadata"]["name"]: d["spec"]["template"]["spec"] for d in documents
            if d["kind"] in ("Deployment", "DaemonSet", "Job")}


def all_env(documents):
    return [e for pod in workloads(documents).values() for c in pod.get("containers", [])
            for e in c.get("env", [])]


def check():
    community = render({})
    assert not any(e["name"] in ("ODIGOS_LICENSE_PROVIDER", "ODIGOS_ONPREM_TOKEN") for e in all_env(community))
    onprem = render({"onPremToken": "render-test-token"})
    assert sum(e["name"] == "ODIGOS_ONPREM_TOKEN" for e in all_env(onprem)) == 4
    assert not any(e["name"] == "ODIGOS_LICENSE_PROVIDER" for e in all_env(onprem))

    marketplace = render(VALUES)
    env = all_env(marketplace)
    assert sum(e["name"] == "ODIGOS_ONPREM_TOKEN" for e in env) == 4
    assert not any(e["name"] == "ODIGOS_LICENSE_PROVIDER" for e in env)
    accounts = [d for d in marketplace if d["kind"] == "ServiceAccount"]
    annotated = [d for d in accounts if d["metadata"].get("annotations", {}).get("eks.amazonaws.com/role-arn")]
    assert len(annotated) == 1 and annotated[0]["metadata"]["name"] == "odigos-instrumentor"
    assert annotated[0]["metadata"]["annotations"]["eks.amazonaws.com/role-arn"] == ROLE
    secret = next(d for d in marketplace if d["kind"] == "Secret" and d["metadata"]["name"] == "odigos-pro")
    assert secret["stringData"] == {"odigos-onprem-token": VALUES["onPremToken"]}
    assert not any(d["kind"] == "Secret" and d.get("type") == "kubernetes.io/dockerconfigjson" for d in marketplace)
    images = {c["image"] for pod in workloads(marketplace).values()
              for c in pod.get("containers", []) + pod.get("initContainers", [])}
    images.update(e["value"] for e in env if e["name"].endswith("_IMAGE") and "value" in e)
    assert images == set(VALUES["images"].values()), images

    structured = copy.deepcopy(VALUES)
    structured["images"] = {c: {"repository": url} for c, url in VALUES["images"].items()}
    assert workloads(render(structured)) == workloads(marketplace)
    regional = copy.deepcopy(structured)
    for entry in regional["images"].values():
        entry["repository"] = entry["repository"].replace("us-east-1", "eu-west-1")
    regional_docs = render(regional)
    regional_env = all_env(regional_docs)
    regional_images = {c["image"] for pod in workloads(regional_docs).values()
                       for c in pod.get("containers", []) + pod.get("initContainers", [])}
    regional_images.update(e["value"] for e in regional_env if e["name"].endswith("_IMAGE") and "value" in e)
    assert regional_images == {url.replace("us-east-1", "eu-west-1") for url in VALUES["images"].values()}
    structured["images"]["enterprise-agents"] = {"repository": ""}
    render(structured, "images.enterprise-agents.repository is required")

    node_values = copy.deepcopy(VALUES)
    node_values["instrumentor"] = {"mountMethod": "k8s-init-container"}
    node_documents = render(node_values)
    agent = next(c for c in workloads(node_documents)["odiglet"]["containers"] if c["name"] == "odiglet")
    assert not any(e["name"].startswith("ODIGOS_MARKETPLACE_") for e in agent["env"])
    manager = next(c for c in workloads(node_documents)["odigos-instrumentor"]["containers"] if c["name"] == "manager")
    pod_uid = next(e for e in manager["env"] if e["name"] == "ODIGOS_MARKETPLACE_POD_UID")
    assert next(e for e in manager["env"] if e["name"] == "ODIGOS_MARKETPLACE_DAEMONSET")["value"] == "odiglet"
    assert pod_uid["valueFrom"]["fieldRef"]["fieldPath"] == "metadata.uid"
    role = next(d for d in node_documents if d["kind"] == "ClusterRole" and d["metadata"]["name"] == "odiglet")
    assert not any("nodes" in rule["resources"] and "update" in rule["verbs"] for rule in role["rules"])
    role = next(d for d in node_documents if d["kind"] == "Role" and d["metadata"]["name"] == "odigos-instrumentor")
    assert any(rule["resources"] == ["configmaps"] and set(rule["verbs"]) == {"get", "list", "create", "update", "delete"} and "resourceNames" not in rule for rule in role["rules"])
    ordinary_role = next(d for d in onprem if d["kind"] == "Role" and d["metadata"]["name"] == "odigos-instrumentor")
    assert not any("configmaps" in rule["resources"] and "create" in rule["verbs"] for rule in ordinary_role["rules"])
    lease_binding = next(d for d in node_documents if d["kind"] == "RoleBinding" and d["metadata"]["name"] == "odigos-instrumentor-leader-election")
    lease_role = next(d for d in node_documents if d["kind"] == "Role" and d["metadata"]["name"] == lease_binding["roleRef"]["name"])
    assert any("leases" in rule["resources"] and {"get", "create", "update"} <= set(rule["verbs"]) for rule in lease_role["rules"])
    custom = copy.deepcopy(VALUES)
    custom["odiglet"] = {"daemonsetName": "custom-odiglet"}
    custom_env = all_env(render(custom))
    assert next(e for e in custom_env if e["name"] == "ODIGOS_MARKETPLACE_DAEMONSET")["value"] == "custom-odiglet"
    node_values["marketplace"]["serviceAccountAnnotations"] = {}
    render(node_values, "Marketplace requires")

    supplied = copy.deepcopy(VALUES)
    supplied["marketplace"]["serviceAccountAnnotations"] = {}
    supplied["marketplace"]["serviceAccountName"] = "aws-marketplace-instrumentor"
    supplied_docs = render(supplied)
    assert workloads(supplied_docs)["odigos-instrumentor"]["serviceAccountName"] == "aws-marketplace-instrumentor"
    assert not any(d["kind"] == "ServiceAccount" and d["metadata"]["name"] in ("odigos-instrumentor", "aws-marketplace-instrumentor") for d in supplied_docs)
    for name in ("odigos-instrumentor", "odigos-instrumentor-leader-election"):
        bindings = [d for d in supplied_docs if d["kind"] in ("RoleBinding", "ClusterRoleBinding") and d["metadata"]["name"] == name]
        assert bindings
        assert all(s["name"] == "aws-marketplace-instrumentor" for d in bindings for s in d["subjects"] if s["kind"] == "ServiceAccount")
    ordinary = render({"onPremToken": "render-test-token", "marketplace": {"serviceAccountName": "ignored-outside-marketplace"}})
    assert workloads(ordinary)["odigos-instrumentor"]["serviceAccountName"] == "odigos-instrumentor"

    values = copy.deepcopy(VALUES)
    del values["onPremToken"]
    render(values, "Marketplace requires an existing Enterprise license")
    values["externalOnpremTokenSecret"] = True
    render(values)
    values = copy.deepcopy(VALUES)
    del values["images"]["enterprise-agents"]
    render(values, "images.enterprise-agents")
    values = copy.deepcopy(VALUES)
    values["insights"] = {"enabled": True}
    values["images"]["enterprise-insights"] = f"{REGISTRY}/odigos-enterprise-insights:test"
    render(values, "Marketplace Insights delivery")
    values = copy.deepcopy(VALUES)
    values["marketplace"]["enabled"] = "false"
    render(values, "marketplace/enabled")
    print("Marketplace, token and community render checks passed")


if __name__ == "__main__":
    check()
