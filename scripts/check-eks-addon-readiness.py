#!/usr/bin/env python3
"""Offline EKS add-on preflight. Requires Helm >=3.19 and PyYAML.

This is a partial readiness check, not AWS certification. It never deploys,
contacts a cluster, or writes rendered Secrets to its report. Source findings
include optional template branches; inspect them before choosing release scope.
"""

import argparse
import collections
import json
from pathlib import Path
import re
import subprocess
import sys

import yaml


def finding(code, message, **context):
    return {"code": code, "message": message, **context}


def scan_templates(chart):
    findings = []
    for path in sorted((chart / "templates").rglob("*")):
        if not path.is_file():
            continue
        text = path.read_text()
        text = re.sub(r"{{-?\s*/\*.*?\*/\s*-?}}",
                      lambda m: "\n" * m[0].count("\n"), text, flags=re.S)
        for match in re.finditer(r"{{.*?}}", text, flags=re.S):
            expression = match[0]
            checks = [
                (r"\blookup\s+", "helm_lookup"),
                (r"\.Release\.(?!Name\b|Namespace\b)\w+", "helm_release_object"),
                (r"\.Capabilities\.APIVersions\b", "review_api_versions"),
                (r"\b(uuidv4|randAlphaNum|genCA|genSignedCert)\b", "review_random_render"),
            ]
            for pattern, code in checks:
                if re.search(pattern, expression):
                    findings.append(finding(code, expression.strip(),
                        file=str(path.relative_to(chart)), line=text.count("\n", 0, match.start()) + 1))
        for number, line in enumerate(text.splitlines(), 1):
            if re.search(r'^\s*["\']?helm\.sh/hook["\']?\s*:', line):
                findings.append(finding("helm_hook", line.strip(),
                    file=str(path.relative_to(chart)), line=number))
    return findings


def walk(value, path=""):
    yield path, value
    if isinstance(value, dict):
        for key, child in value.items():
            yield from walk(child, f"{path}/{key}")
    elif isinstance(value, list):
        for index, child in enumerate(value):
            yield from walk(child, f"{path}/{index}")


def resource_id(doc):
    metadata = doc.get("metadata", {})
    return f'{doc.get("kind")}/{metadata.get("namespace", "")}/{metadata.get("name", "")}'


def inspect_render(documents, registry):
    images = collections.defaultdict(list)
    findings = []
    crds = {(d["spec"]["group"], d["spec"]["names"]["kind"])
            for d in documents if d.get("kind") == "CustomResourceDefinition"}
    for doc in documents:
        identity = resource_id(doc)
        group = doc.get("apiVersion", "").split("/")[0]
        if (group, doc.get("kind")) in crds:
            findings.append(finding("crd_and_custom_resource", identity))
        for path, value in walk(doc.get("spec", {})):
            if not isinstance(value, dict):
                continue
            image = value.get("image")
            if isinstance(image, str):
                images[image].append(f"{identity}/spec{path}/image")
            if str(value.get("name", "")).endswith("_IMAGE") and isinstance(value.get("value"), str):
                images[value["value"]].append(f"{identity}/spec{path}/value")
    if not any(d.get("kind") in ("Deployment", "DaemonSet") for d in documents):
        findings.append(finding("missing_workload", "No Deployment or DaemonSet rendered"))
    for image in sorted(images):
        if not image.startswith(registry + "/"):
            findings.append(finding("image_outside_marketplace", image))
        if image.endswith(":v0.0.0") or image.endswith(":0.0.0"):
            findings.append(finding("placeholder_image_tag", image))
    return dict(sorted(images.items())), findings


def run(command):
    return subprocess.run(command, text=True, capture_output=True, timeout=120)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("chart", type=Path)
    parser.add_argument("--kube-version", required=True, help="Render target only; not a certification claim")
    parser.add_argument("--values", action="append", default=[])
    parser.add_argument("--namespace", default="odigos-system")
    parser.add_argument("--registry", default="709825985650.dkr.ecr.us-east-1.amazonaws.com")
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    chart = args.chart.resolve()
    report = {"chart": str(chart), "kubernetesRenderTarget": args.kube_version,
              "limitations": ["No image pulls, vulnerability scans, AWS API calls, licensing checks, or runtime tests",
                              "Source scan includes inactive branches and is not a complete Go-template parser",
                              "Image inventory includes workload images and *_IMAGE environment values; review dynamic code paths"],
              "findings": scan_templates(chart)}
    version = run(["helm", "version", "--short"])
    report["helmVersion"] = version.stdout.strip()
    match = re.search(r"v(\d+)\.(\d+)\.(\d+)", version.stdout)
    if version.returncode or not match or tuple(map(int, match.groups())) < (3, 19, 0):
        raise SystemExit("Helm >=3.19 is required")
    schema = chart / "aws_mp_configuration_schema.json"
    if not schema.exists():
        report["findings"].append(finding("missing_addon_schema", schema.name))
    else:
        try:
            document = json.loads(schema.read_text())
            report["addonSchemaDialect"] = document.get("$schema")
        except json.JSONDecodeError:
            report["findings"].append(finding("invalid_addon_schema", "Invalid JSON"))
    report["podIdentityParametersPresent"] = (chart / "aws_mp_addon_parameters.json").exists()
    flags = [arg for value in args.values for arg in ("--values", value)]
    lint = run(["helm", "lint", str(chart), *flags])
    report["helmLint"] = {"passed": lint.returncode == 0}
    if lint.returncode:
        report["findings"].append(finding("helm_lint_failed", "Run helm lint locally for details"))
    command = ["helm", "template", "odigos", str(chart), "--namespace", args.namespace,
               "--kube-version", args.kube_version, "--include-crds", "--no-hooks", *flags]
    rendered = run(command)
    report["helmTemplate"] = {"passed": rendered.returncode == 0}
    if rendered.returncode:
        report["findings"].append(finding("helm_template_failed", "Run helm template locally for details"))
    else:
        documents = [d for d in yaml.safe_load_all(rendered.stdout) if isinstance(d, dict)]
        report["resourceCounts"] = dict(sorted(collections.Counter(d.get("kind") for d in documents).items()))
        report["images"], findings = inspect_render(documents, args.registry)
        report["findings"].extend(findings)
        again = run(command)
        if again.returncode:
            report["findings"].append(finding("second_render_failed", "Repeat helm template failed"))
        else:
            second = {resource_id(d): d for d in yaml.safe_load_all(again.stdout) if isinstance(d, dict)}
            first = {resource_id(d): d for d in documents}
            changed = sorted(k for k in first.keys() | second.keys() if first.get(k) != second.get(k))
            report["resourcesChangingBetweenIdenticalRenders"] = changed
            if changed:
                report["findings"].append(finding("review_unstable_render", ", ".join(changed)))
    report["findingCounts"] = dict(sorted(collections.Counter(f["code"] for f in report["findings"]).items()))
    report["status"] = "needs_work_or_review" if report["findings"] else "local_checks_passed_not_certified"
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({"status": report["status"], "findingCounts": report["findingCounts"],
                      "report": str(args.output)}, indent=2))
    return 1 if report["findings"] else 0


if __name__ == "__main__":
    sys.exit(main())
