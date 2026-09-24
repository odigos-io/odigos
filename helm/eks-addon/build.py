#!/usr/bin/env python3
"""Build an offline EKS submission candidate from the regular chart; does not publish."""

import argparse
import json
from pathlib import Path
import re
import shutil
import subprocess

import yaml

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[1]
COMPONENTS = {"autoscaler", "scheduler", "enterprise-instrumentor", "enterprise-odiglet",
              "enterprise-collector", "enterprise-ui", "enterprise-agents", "cli"}
REGISTRY = "709825985650.dkr.ecr.us-east-1.amazonaws.com/"


def replace_once(path, old, new):
    source = path.read_text()
    if source.count(old) != 1:
        raise ValueError(f"source changed; review EKS adaptation in {path.name}")
    path.write_text(source.replace(old, new, 1))


def replace_helper(path, name, body):
    source = path.read_text()
    pattern = r'(?ms)^\{\{- define "' + re.escape(name) + r'" -\}\}.*?(?=^\{\{- define ")'
    source, count = re.subn(pattern, '{{- define "' + name + '" -}}\n' + body + '\n{{- end -}}\n\n', source)
    if count != 1:
        raise ValueError(f"source changed; review helper {name}")
    path.write_text(source)


def run(*args):
    return subprocess.run(args, capture_output=True, text=True, check=True, timeout=120).stdout


def build(output, version, images, namespace="odigos-system", kube_version="1.34.0"):
    if not re.fullmatch(r"[1-9][0-9]*\.[0-9]+\.[0-9]+", version):
        raise ValueError("provide the selected stable release version without v; no prerelease or placeholder")
    if set(images) != COMPONENTS:
        raise ValueError(f"images must contain exactly: {', '.join(sorted(COMPONENTS))}")
    for component, uri in images.items():
        if not isinstance(uri, str) or not re.fullmatch(re.escape(REGISTRY) + r"[a-z0-9/_-]+(?::[A-Za-z0-9_.-]+|@sha256:[0-9a-f]{64})", uri):
            raise ValueError(f"{component} needs an explicit Marketplace ECR image tag or digest")
        if uri.endswith((":latest", ":v0.0.0", ":0.0.0")):
            raise ValueError(f"{component} uses an unversioned or placeholder tag")
    if not re.fullmatch(r"[a-z0-9](?:[-a-z0-9]{0,61}[a-z0-9])?", namespace) or namespace in ("default", "kube-system", "kube-public", "kube-node-lease"):
        raise ValueError("use a dedicated Kubernetes namespace")
    output = output.resolve()
    if output.exists():
        raise ValueError("output must not already exist")
    chart = output / "odigos"
    shutil.copytree(ROOT / "helm/odigos", chart)
    for name in ("insights", "gke", "centralproxy"):
        shutil.rmtree(chart / "templates" / name, ignore_errors=True)
    (chart / "templates/cleanup/cleanup-job.yaml").unlink()
    (chart / "templates/NOTES.txt").write_text("Use the EKS add-on lifecycle instructions shipped with this release.\n")
    for path in (HERE / "overlays").rglob("*"):
        if path.is_file():
            destination = chart / "templates" / path.relative_to(HERE / "overlays")
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(path, destination)
    helpers = chart / "templates/_helpers.tpl"
    replace_helper(helpers, "utils.imagePrefix", '{{ regexReplaceAll "/[^/]+$" .Values.images.autoscaler "" }}')
    replace_helper(helpers, "odigos.onPremToken", "")
    replace_helper(helpers, "odigos.secretExists", "true")
    replace_once(chart / "templates/scheduler/deployment.yaml", "        env:\n",
                 '        env:\n          - name: ODIGOS_INSTALLATION_METHOD\n            value: eks-addon\n')
    replace_once(chart / "templates/odiglet/daemonset.yaml", "            name: odigos-go-offsets\n",
                 "            name: odigos-go-offsets\n            items:\n              - key: go_offset_results.json\n                path: go_offset_results.json\n")
    metadata = yaml.safe_load((chart / "Chart.yaml").read_text())
    metadata.update(version=version, appVersion="v" + version)
    (chart / "Chart.yaml").write_text(yaml.safe_dump(metadata, sort_keys=False))
    values = yaml.load((chart / "values.yaml").read_text(), Loader=yaml.CSafeLoader)
    values["images"] = images
    values["marketplace"]["enabled"] = True
    values["nodeSelector"] = {"kubernetes.io/os": "linux"}
    values["imagePrefix"] = ""
    (chart / "values.yaml").write_text(yaml.safe_dump(values, sort_keys=False))
    for name in ("aws_mp_configuration_schema.json", "aws_mp_addon_parameters.json"):
        shutil.copyfile(HERE / name, chart / name)

    # The cleanup job is an explicit pre-delete operation, never installed by EKS.
    cleanup_values = output / "cleanup-values.json"
    cleanup_values.write_text(json.dumps({"marketplace": {"enabled": True}, "images": images}))
    cleanup = run("helm", "template", "odigos", str(ROOT / "helm/odigos"), "--namespace", namespace,
                  "--values", str(cleanup_values), "--show-only", "templates/cleanup/cleanup-job.yaml")
    job = yaml.safe_load(cleanup)
    job["metadata"].pop("annotations", None)
    job["spec"].pop("ttlSecondsAfterFinished", None)
    (output / "cleanup-job.yaml").write_text(yaml.safe_dump(job, sort_keys=False))
    cleanup_values.unlink()

    report_path = output / "preflight.json"
    run("python3", str(ROOT / "scripts/check-eks-addon-readiness.py"), str(chart),
        "--namespace", namespace, "--kube-version", kube_version, "--output", str(report_path))
    report = json.loads(report_path.read_text())
    if set(report["images"]) != set(images.values()) - {images["cli"]}:
        raise ValueError("rendered image inventory differs from declared core release images")
    run("helm", "package", str(chart), "--destination", str(output))
    (output / "release-manifest.json").write_text(json.dumps({
        "version": version, "namespace": namespace, "images": images,
        "kubernetesRenderTarget": kube_version,
        "status": "offline-candidate-not-runtime-verified-or-certified",
    }, indent=2) + "\n")
    return chart


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True)
    parser.add_argument("--images", required=True, type=Path, help="JSON mapping of component names to Marketplace ECR image references")
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--namespace", default="odigos-system")
    parser.add_argument("--kube-version", default="1.34.0", help="Offline render target, not a support claim")
    args = parser.parse_args()
    print(build(args.output, args.version, json.loads(args.images.read_text()), args.namespace, args.kube_version))
