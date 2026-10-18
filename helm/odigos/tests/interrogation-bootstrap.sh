#!/usr/bin/env bash
set -euo pipefail

chart_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bootstrap_template=templates/insights/interrogation/litellm/bootstrap-configmap.yaml
deployment_template=templates/insights/interrogation/litellm/deployment.yaml
flags=(--set insights.enabled=true --set insights.interrogation.enabled=true --set onPremToken=fixture-token)

python3 - "$chart_dir/values.schema.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as file:
    schema = json.load(file)
litellm = schema["properties"]["insights"]["properties"]["interrogation"]["properties"]["litellm"]
assert "bootstrapConfig" in litellm["properties"]
assert "bootstrapConfig" not in litellm.get("required", [])
PY

if helm template odigos "$chart_dir" "${flags[@]}" --show-only "$bootstrap_template" >/dev/null 2>&1; then
  echo 'bootstrap ConfigMap rendered without bootstrapConfig=true' >&2
  exit 1
fi

if helm template odigos "$chart_dir" --set insights.enabled=true --set insights.interrogation.enabled=true \
  --set insights.interrogation.litellm.bootstrapConfig=true --show-only "$bootstrap_template" >/dev/null 2>&1; then
  echo 'bootstrap ConfigMap rendered without an enterprise token' >&2
  exit 1
fi

bootstrap="$(helm template odigos "$chart_dir" "${flags[@]}" \
  --set insights.interrogation.litellm.bootstrapConfig=true --show-only "$bootstrap_template")"
grep -q 'helm.sh/resource-policy: keep' <<<"$bootstrap"
grep -q 'name: odigos-litellm' <<<"$bootstrap"
grep -q 'model_name: interrogation' <<<"$bootstrap"
grep -q 'master_key: os.environ/LITELLM_MASTER_KEY' <<<"$bootstrap"

deployment="$(helm template odigos "$chart_dir" "${flags[@]}" --show-only "$deployment_template")"
grep -q 'name: odigos-litellm-runtime' <<<"$deployment"
if grep -q 'optional: true' <<<"$deployment"; then
  echo 'proxy runtime key reference must be required' >&2
  exit 1
fi

echo 'Interrogation bootstrap render fixtures passed'
