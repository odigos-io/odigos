#!/usr/bin/env bash
set -euo pipefail

# Accept optional path argument
P=${1:-"../../.."}

echo "🚀 Rendering Helm template for odiglet DaemonSet..."
helm template odigos "$P/helm/odigos" \
  --set collectorNode.limitMemoryMiB=570 \
  --set collectorNode.limitCPUm=560 \
  --show-only templates/odiglet/daemonset.yaml > /tmp/helm-template-output.yaml

YAML_FILE="/tmp/helm-template-output.yaml"
DC='.spec.template.spec.containers[] | select(.name=="data-collection") | .resources'

echo "📋 Extracting data-collection container resources..."
yq "$DC" "$YAML_FILE"
echo

MEMORY_REQUEST=$(yq "$DC.requests.memory" "$YAML_FILE")
MEMORY_LIMIT=$(yq "$DC.limits.memory" "$YAML_FILE")
CPU_REQUEST=$(yq "$DC.requests.cpu" "$YAML_FILE")
CPU_LIMIT=$(yq "$DC.limits.cpu" "$YAML_FILE")

printf "🔍 Values:\n"
printf "  memory: request=[%s] limit=[%s]\n" "$MEMORY_REQUEST" "$MEMORY_LIMIT"
printf "  cpu:    request=[%s] limit=[%s]\n" "$CPU_REQUEST" "$CPU_LIMIT"
printf "\n"

echo "✅ Verifying mirroring (requests should equal limits)..."

rc=0
if [[ "$MEMORY_REQUEST" == "570Mi" && "$MEMORY_LIMIT" == "570Mi" ]]; then
  echo "✅ Memory mirroring works: request=$MEMORY_REQUEST, limit=$MEMORY_LIMIT"
else
  echo "❌ Memory mirroring failed: request='$MEMORY_REQUEST', limit='$MEMORY_LIMIT' (expected both '570Mi')"
  rc=1
fi

if [[ "$CPU_REQUEST" == "560m" && "$CPU_LIMIT" == "560m" ]]; then
  echo "✅ CPU mirroring works: request=$CPU_REQUEST, limit=$CPU_LIMIT"
else
  echo "❌ CPU mirroring failed: request='$CPU_REQUEST', limit='$CPU_LIMIT' (expected both '560m')"
  rc=1
fi

exit $rc
