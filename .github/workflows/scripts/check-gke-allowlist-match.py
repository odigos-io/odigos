#!/usr/bin/env python3
"""Check that a rendered odiglet DaemonSet matches a GKE WorkloadAllowlist.

Implements the WorkloadAllowlist matching rules documented at:
https://cloud.google.com/kubernetes-engine/docs/reference/crds/workloadallowlist

Succeeds if any allowlist YAML under --allowlists-dir matches the DaemonSet
pod template. Prints mismatches for each candidate when none match.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path
from typing import Any

import yaml


def load_yaml_docs(path: Path) -> list[Any]:
    with path.open() as f:
        return [doc for doc in yaml.safe_load_all(f) if doc is not None]


def strip_image_ref(image: str) -> str:
    """Remove tag or digest; WorkloadAllowlist image matches are without them."""
    if "@" in image:
        return image.split("@", 1)[0]
    # Keep registry ports (host:port/path) intact; only strip a trailing :tag.
    if image.count(":") == 1 and "/" in image.split(":", 1)[0]:
        # unlikely host-only
        pass
    name, _, maybe_tag = image.rpartition(":")
    if name and "/" in name and maybe_tag and "/" not in maybe_tag and "@" not in maybe_tag:
        return name
    return image


def is_regex(pattern: str) -> bool:
    return pattern.startswith("^") and pattern.endswith("$")


def match_string(workload: str, allowlist: str) -> bool:
    if is_regex(allowlist):
        return re.fullmatch(allowlist[1:-1], workload) is not None
    return workload == allowlist


def match_string_list(workload_vals: list[str] | None, allowlist_vals: list[str] | None, field: str) -> list[str]:
    """Every workload value must match some allowlist entry (allowlist is a superset)."""
    errors: list[str] = []
    if allowlist_vals is None:
        return errors
    workload_vals = workload_vals or []
    for wv in workload_vals:
        if not any(match_string(wv, av) for av in allowlist_vals):
            errors.append(f"{field}: workload value {wv!r} does not match any allowlist entry {allowlist_vals!r}")
    return errors


def match_env(workload_envs: list[dict] | None, allowlist_envs: list[dict] | None, prefix: str) -> list[str]:
    errors: list[str] = []
    if allowlist_envs is None:
        return errors
    patterns = [e.get("name", "") for e in allowlist_envs]
    for env in workload_envs or []:
        name = env.get("name", "")
        if not any(match_string(name, p) for p in patterns):
            errors.append(f"{prefix}.env: {name!r} not allowed (allowlist: {patterns!r})")
    return errors


def match_env_from(workload: list[dict] | None, allowlist: list[dict] | None, prefix: str) -> list[str]:
    errors: list[str] = []
    if allowlist is None:
        return errors
    allowed_cms = {
        e["configMapRef"]["name"]
        for e in allowlist
        if isinstance(e.get("configMapRef"), dict) and "name" in e["configMapRef"]
    }
    allowed_secrets = {
        e["secretRef"]["name"]
        for e in allowlist
        if isinstance(e.get("secretRef"), dict) and "name" in e["secretRef"]
    }
    for entry in workload or []:
        cm = entry.get("configMapRef") or {}
        sec = entry.get("secretRef") or {}
        if "name" in cm and cm["name"] not in allowed_cms:
            errors.append(f"{prefix}.envFrom.configMapRef: {cm['name']!r} not in allowlist {sorted(allowed_cms)!r}")
        if "name" in sec and sec["name"] not in allowed_secrets:
            errors.append(f"{prefix}.envFrom.secretRef: {sec['name']!r} not in allowlist {sorted(allowed_secrets)!r}")
    return errors


def match_volume_mount(wvm: dict, avm: dict) -> bool:
    if wvm.get("name") != avm.get("name"):
        return False
    if wvm.get("mountPath") != avm.get("mountPath"):
        return False
    if "subPath" in avm and wvm.get("subPath") != avm.get("subPath"):
        return False
    # If allowlist omits readOnly, any workload value is accepted for this field.
    # If allowlist sets readOnly, workload must match (omitted workload readOnly == false).
    if "readOnly" in avm:
        w_ro = bool(wvm.get("readOnly", False))
        a_ro = bool(avm["readOnly"])
        if w_ro != a_ro:
            return False
    return True


def match_volume_mounts(workload: list[dict] | None, allowlist: list[dict] | None, prefix: str) -> list[str]:
    errors: list[str] = []
    if allowlist is None:
        return errors
    for wvm in workload or []:
        if not any(match_volume_mount(wvm, avm) for avm in allowlist):
            errors.append(
                f"{prefix}.volumeMounts: no allowlist match for "
                f"name={wvm.get('name')!r} mountPath={wvm.get('mountPath')!r} readOnly={wvm.get('readOnly', False)}"
            )
    return errors


def match_probe(workload: dict | None, allowlist: dict | None, prefix: str) -> list[str]:
    errors: list[str] = []
    if allowlist is None:
        return errors
    a_exec = (allowlist.get("exec") or {}).get("command")
    if a_exec is None:
        return errors
    w_exec = ((workload or {}).get("exec") or {}).get("command") or []
    if w_exec != a_exec:
        errors.append(f"{prefix}: probe exec.command workload={w_exec!r} allowlist={a_exec!r}")
    return errors


def match_security_context(workload: dict | None, allowlist: dict | None, prefix: str) -> list[str]:
    errors: list[str] = []
    if allowlist is None:
        return errors
    workload = workload or {}
    w_priv = bool(workload.get("privileged", False))
    # GKE: set privileged: true iff the workload is privileged; otherwise omit the field.
    if "privileged" in allowlist:
        if bool(allowlist["privileged"]) != w_priv:
            errors.append(
                f"{prefix}.securityContext.privileged: "
                f"workload={w_priv} allowlist={bool(allowlist['privileged'])}"
            )
    elif w_priv:
        errors.append(
            f"{prefix}.securityContext.privileged: workload=True but allowlist omits privileged (must be true)"
        )
    a_caps = allowlist.get("capabilities") or {}
    w_caps = workload.get("capabilities") or {}
    if "add" in a_caps or w_caps.get("add"):
        # If workload adds capabilities, allowlist must list them (workload ⊆ allowlist).
        if "add" not in a_caps and w_caps.get("add"):
            errors.append(
                f"{prefix}.securityContext.capabilities.add: "
                f"workload has {w_caps.get('add')!r} but allowlist omits capabilities.add"
            )
        else:
            for cap in w_caps.get("add") or []:
                if cap not in (a_caps.get("add") or []):
                    errors.append(f"{prefix}.securityContext.capabilities.add: {cap!r} not in allowlist")
    if "drop" in a_caps or w_caps.get("drop"):
        if "drop" not in a_caps and w_caps.get("drop"):
            errors.append(
                f"{prefix}.securityContext.capabilities.drop: "
                f"workload has {w_caps.get('drop')!r} but allowlist omits capabilities.drop"
            )
        else:
            for cap in w_caps.get("drop") or []:
                if cap not in (a_caps.get("drop") or []):
                    errors.append(f"{prefix}.securityContext.capabilities.drop: {cap!r} not in allowlist")
    return errors


def match_container(workload: dict, allowlist: dict, kind: str) -> list[str]:
    name = workload.get("name")
    prefix = f"{kind}/{name}"
    errors: list[str] = []

    a_image = allowlist.get("image")
    if a_image is not None:
        w_image = strip_image_ref(workload.get("image", ""))
        if not match_string(w_image, a_image):
            errors.append(f"{prefix}.image: {w_image!r} does not match {a_image!r}")

    errors.extend(match_string_list(workload.get("command"), allowlist.get("command"), f"{prefix}.command"))
    errors.extend(match_string_list(workload.get("args"), allowlist.get("args"), f"{prefix}.args"))
    errors.extend(match_env(workload.get("env"), allowlist.get("env"), prefix))
    errors.extend(match_env_from(workload.get("envFrom"), allowlist.get("envFrom"), prefix))
    errors.extend(match_security_context(workload.get("securityContext"), allowlist.get("securityContext"), prefix))
    errors.extend(match_volume_mounts(workload.get("volumeMounts"), allowlist.get("volumeMounts"), prefix))
    errors.extend(match_probe(workload.get("livenessProbe"), allowlist.get("livenessProbe"), f"{prefix}.livenessProbe"))
    errors.extend(match_probe(workload.get("readinessProbe"), allowlist.get("readinessProbe"), f"{prefix}.readinessProbe"))
    errors.extend(match_probe(workload.get("startupProbe"), allowlist.get("startupProbe"), f"{prefix}.startupProbe"))
    return errors


def match_volume(wv: dict, av: dict) -> bool:
    if wv.get("name") != av.get("name"):
        return False
    if "hostPath" in av:
        w_hp = wv.get("hostPath") or {}
        a_hp = av["hostPath"] or {}
        if "path" in a_hp and w_hp.get("path") != a_hp.get("path"):
            return False
        return "hostPath" in wv
    if "configMap" in av:
        w_cm = wv.get("configMap") or {}
        a_cm = av["configMap"] or {}
        if "name" in a_cm and w_cm.get("name") != a_cm.get("name"):
            return False
        return "configMap" in wv
    if "emptyDir" in av:
        return "emptyDir" in wv
    if "secret" in av:
        return "secret" in wv
    # Name-only allowlist entry (e.g. emptyDir exchange-dir listed by name alone).
    return True


def match_volumes(workload: list[dict] | None, allowlist: list[dict] | None) -> list[str]:
    errors: list[str] = []
    if allowlist is None:
        return errors
    for wv in workload or []:
        if not any(match_volume(wv, av) for av in allowlist):
            errors.append(f"volumes: no allowlist match for {wv.get('name')!r} keys={sorted(k for k in wv if k != 'name')}")
    return errors


def index_by_name(items: list[dict] | None) -> dict[str, dict]:
    return {c["name"]: c for c in (items or []) if "name" in c}


def match_required_true_flag(workload_true: bool, allowlist: dict, field: str, prefix: str = "") -> list[str]:
    """GKE rule: set field to true in the allowlist iff the workload sets it to true."""
    errors: list[str] = []
    label = f"{prefix}{field}" if prefix else field
    if field in allowlist:
        if bool(allowlist[field]) != workload_true:
            errors.append(f"{label}: workload={workload_true} allowlist={bool(allowlist[field])}")
    elif workload_true:
        errors.append(f"{label}: workload=True but allowlist omits {field} (must be true)")
    return errors


def match_pod_to_allowlist(pod_spec: dict, criteria: dict) -> list[str]:
    errors: list[str] = []

    for field in ("hostPID", "hostNetwork", "hostIPC"):
        errors.extend(
            match_required_true_flag(bool(pod_spec.get(field, False)), criteria, field)
        )

    for kind, pod_key, al_key in (
        ("initContainer", "initContainers", "initContainers"),
        ("container", "containers", "containers"),
    ):
        al_by_name = index_by_name(criteria.get(al_key))
        for wc in pod_spec.get(pod_key) or []:
            name = wc.get("name")
            if name not in al_by_name:
                errors.append(f"{kind}/{name}: not present in allowlist {al_key}")
                continue
            errors.extend(match_container(wc, al_by_name[name], kind))

    errors.extend(match_volumes(pod_spec.get("volumes"), criteria.get("volumes")))
    return errors


def extract_daemonset_pod_spec(docs: list[Any], name: str = "odiglet") -> dict:
    for doc in docs:
        if doc.get("kind") == "DaemonSet" and doc.get("metadata", {}).get("name") == name:
            return doc["spec"]["template"]["spec"]
    raise SystemExit(f"DaemonSet named {name!r} not found in rendered manifests")


def find_allowlists(allowlists_dir: Path) -> list[Path]:
    return sorted(allowlists_dir.rglob("*.yaml")) + sorted(allowlists_dir.rglob("*.yml"))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--daemonset", required=True, type=Path, help="Rendered helm output containing the odiglet DaemonSet")
    parser.add_argument("--allowlists-dir", required=True, type=Path, help="Directory of WorkloadAllowlist YAML files")
    parser.add_argument("--daemonset-name", default="odiglet")
    args = parser.parse_args()

    pod_spec = extract_daemonset_pod_spec(load_yaml_docs(args.daemonset), args.daemonset_name)
    allowlist_paths = find_allowlists(args.allowlists_dir)
    if not allowlist_paths:
        print(f"ERROR: no allowlist YAML files under {args.allowlists_dir}", file=sys.stderr)
        return 1

    all_failures: list[str] = []
    for path in allowlist_paths:
        docs = load_yaml_docs(path)
        for doc in docs:
            if doc.get("kind") != "WorkloadAllowlist":
                continue
            name = doc.get("metadata", {}).get("name", path.name)
            criteria = doc.get("matchingCriteria") or {}
            errors = match_pod_to_allowlist(pod_spec, criteria)
            if not errors:
                print(f"MATCH: DaemonSet/{args.daemonset_name} matches WorkloadAllowlist {name!r} ({path})")
                return 0
            all_failures.append(f"--- {path} ({name}) ---")
            all_failures.extend(f"  - {e}" for e in errors)

    print("ERROR: DaemonSet did not match any WorkloadAllowlist", file=sys.stderr)
    print("\n".join(all_failures), file=sys.stderr)
    return 1


if __name__ == "__main__":
    sys.exit(main())
