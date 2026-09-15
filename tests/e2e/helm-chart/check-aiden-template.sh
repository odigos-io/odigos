#!/usr/bin/env bash
# Helm template smoke test for Aiden resources (placeholder keys in a temp values file).
set -euo pipefail

P="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
values_file="$(mktemp)"
rendered="$(mktemp)"
trap 'rm -f "$values_file" "$rendered"' EXIT

cat >"$values_file" <<'EOF'
onPremToken: test-token
aiden:
  enabled: true
  gemini:
    key: placeholder-gemini-key
  jaeger:
    endpoints:
      default: http://jaeger-query.tracing.svc:16686
EOF

# Slack is optional: in-UI chat is the default path when Aiden is enabled.
helm template odigos "$P/helm/odigos" \
  --namespace odigos-system \
  -f "$values_file" \
  --show-only templates/aiden/configmap.yaml \
  --show-only templates/aiden/deployment.yaml \
  --show-only templates/aiden/service.yaml \
  --show-only templates/aiden/secret.yaml \
  --show-only templates/ui/deployment.yaml \
  >"$rendered"

grep -q 'jaeger-config.json' "$rendered"
grep -q 'JAEGER_SKILL_CONFIG' "$rendered"
grep -q '/app/aiden-skills' "$rendered"
grep -q '/app/aiden/' "$rendered"
grep -q 'AGENTS.md' "$rendered"
grep -q 'OPENCLAW_GATEWAY_TOKEN' "$rendered"
grep -q 'AIDEN_GATEWAY_URL' "$rendered"
grep -q 'kind: Service' "$rendered"
grep -q 'name: odigos-aiden' "$rendered"
grep -q '"bind": "loopback"' "$rendered"
grep -q '"port": 18790' "$rendered"
# Slack channel must not be configured when tokens are omitted.
if grep -q '"slack"' "$rendered"; then
  echo "expected no Slack channel when slack tokens are empty" >&2
  exit 1
fi
if grep -q 'SLACK_APP_TOKEN' "$rendered"; then
  echo "expected no Slack env when slack tokens are empty" >&2
  exit 1
fi

# Interrogation without Slack: cron targets the UI main session, no Slack announce.
cat >"$values_file" <<'EOF'
onPremToken: test-token
aiden:
  enabled: true
  interrogation:
    enabled: true
  gemini:
    key: placeholder-gemini-key
  jaeger:
    endpoints:
      default: http://jaeger-query.tracing.svc:16686
EOF

helm template odigos "$P/helm/odigos" \
  --namespace odigos-system \
  -f "$values_file" \
  --show-only templates/aiden/deployment.yaml \
  >"$rendered"

grep -q -- '--session main' "$rendered"
grep -q -- '--name odigos-interrogation' "$rendered"
if grep -q -- '--channel slack' "$rendered"; then
  echo "expected no Slack announce when interrogation is on and Slack is omitted" >&2
  exit 1
fi

# interrogationTarget without Slack tokens must fail.
if helm template odigos "$P/helm/odigos" \
  --namespace odigos-system \
  -f "$values_file" \
  --set aiden.slack.interrogationTarget='channel:C0123456789' \
  --show-only templates/aiden/deployment.yaml \
  >/dev/null 2>"$rendered"; then
  echo "expected helm template to fail when interrogationTarget is set without Slack tokens" >&2
  exit 1
fi
grep -q 'interrogationTarget is set' "$rendered"

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
  >"$rendered"

grep -q 'jaeger-config.json' "$rendered"
grep -q 'SLACK_APP_TOKEN' "$rendered"
grep -q '"slack"' "$rendered"

# Interrogation with Slack: UI session plus Slack announce of the same reply.
cat >"$values_file" <<'EOF'
onPremToken: test-token
aiden:
  enabled: true
  interrogation:
    enabled: true
  gemini:
    key: placeholder-gemini-key
  slack:
    key: xapp-placeholder
    botToken: xoxb-placeholder
    interrogationTarget: channel:C0123456789
  jaeger:
    endpoints:
      default: http://jaeger-query.tracing.svc:16686
EOF

helm template odigos "$P/helm/odigos" \
  --namespace odigos-system \
  -f "$values_file" \
  --show-only templates/aiden/deployment.yaml \
  >"$rendered"

grep -q -- '--session main' "$rendered"
grep -q -- '--announce' "$rendered"
grep -q -- '--channel slack' "$rendered"
grep -q -- 'channel:C0123456789' "$rendered"

echo "Aiden helm template checks passed"
