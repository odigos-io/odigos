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
    "marketplace": {"enabled": True, "licenseManagerRegion": "us-east-1",
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
    assert not any(e["name"] == "ODIGOS_ONPREM_TOKEN" for e in env)
    assert sum(e["name"] == "ODIGOS_LICENSE_PROVIDER" and e.get("value") == "aws-marketplace" for e in env) == 5
    assert sum(e["name"] == "ODIGOS_MARKETPLACE_REGION" and e.get("value") == "us-east-1" for e in env) == 5
    for name in ("odigos-instrumentor", "odigos-ui", "odiglet", "odigos-gateway"):
        account = next(d for d in marketplace if d["kind"] == "ServiceAccount" and d["metadata"]["name"] == name)
        assert account["metadata"]["annotations"]["eks.amazonaws.com/role-arn"] == ROLE
    secret = next(d for d in marketplace if d["kind"] == "Secret" and d["metadata"]["name"] == "odigos-pro")
    assert secret["stringData"] == {"aws-marketplace": "true"}
    assert not any(d["kind"] == "Secret" and d.get("type") == "kubernetes.io/dockerconfigjson" for d in marketplace)
    images = {c["image"] for pod in workloads(marketplace).values()
              for c in pod.get("containers", []) + pod.get("initContainers", [])}
    images.update(e["value"] for e in env if e["name"].endswith("_IMAGE") and "value" in e)
    assert images == set(VALUES["images"].values()), images

    for key in ("onPremToken", "externalOnpremTokenSecret", "externalOnpremPullSecret"):
        values = copy.deepcopy(VALUES)
        values[key] = "render-test-token" if key == "onPremToken" else True
        render(values, "cannot be combined")
    values = copy.deepcopy(VALUES)
    del values["images"]["enterprise-agents"]
    render(values, "images.enterprise-agents")
    values = copy.deepcopy(VALUES)
    values["marketplace"]["licenseManagerRegion"] = ""
    render(values, "licenseManagerRegion is required")
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
