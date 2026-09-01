#!/usr/bin/env bash
# Helm template smoke test for Aiden resources (placeholder keys in a temp values file).
set -euo pipefail

P="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
values_file="$(mktemp)"
trap 'rm -f "$values_file"' EXIT

cat >"$values_file" <<'EOF'
onPremToken: test-token
aiden:
  enabled: true
  gemini:
    key: placeholder-gemini-key
  slack:
    key: xapp-placeholder
    botToken: xoxb-placeholder
  jaeger:
    endpoints:
      default: http://jaeger-query.tracing.svc:16686
EOF

helm template odigos "$P/helm/odigos" \
  --namespace odigos-system \
  -f "$values_file" \
  --show-only templates/aiden/configmap.yaml \
  --show-only templates/aiden/deployment.yaml \
  >/tmp/helm-aiden-template.yaml

grep -q 'jaeger-config.json' /tmp/helm-aiden-template.yaml
grep -q 'JAEGER_SKILL_CONFIG' /tmp/helm-aiden-template.yaml
grep -q '/app/aiden-skills' /tmp/helm-aiden-template.yaml
grep -q '/app/aiden/AGENTS.md' /tmp/helm-aiden-template.yaml
grep -q 'jaeger' /tmp/helm-aiden-template.yaml

echo "Aiden helm template checks passed"
