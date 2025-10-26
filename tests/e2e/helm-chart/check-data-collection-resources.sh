#!/usr/bin/env bash
set -euo pipefail

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

echo "📋 Extracting data-collection container resources..."

RESOURCES=$(grep -A 60 "name: data-collection" "$YAML_FILE" | grep -A 15 "resources:" || true)

echo "🔍 Found resources section:"
echo "$RESOURCES"
echo

# --- Extraction ---
MEMORY_REQUEST_RAW=$(echo "$RESOURCES" | grep -A 2 "requests:" | grep "memory:" | awk '{print $2}')
MEMORY_LIMIT_RAW=$(echo "$RESOURCES" | grep -A 2 "limits:" | grep "memory:" | awk '{print $2}')

# --- Normalization helper ---
normalize() {
  local val="$1"
  # Remove quotes, carriage returns, newlines, and extra spaces
  echo "$val" | tr -d '\r' | tr -d '\n' | sed 's/"//g' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//'
}

MEMORY_REQUEST=$(normalize "$MEMORY_REQUEST_RAW")
MEMORY_LIMIT=$(normalize "$MEMORY_LIMIT_RAW")

printf "🔍 Debug raw values:\n"
printf "  MEMORY_REQUEST_RAW=[%s]\n" "$MEMORY_REQUEST_RAW"
printf "  MEMORY_LIMIT_RAW=[%s]\n" "$MEMORY_LIMIT_RAW"
printf "\n"

printf "🔍 Normalized values:\n"
printf "  MEMORY_REQUEST=[%s]\n" "$MEMORY_REQUEST"
printf "  MEMORY_LIMIT=[%s]\n" "$MEMORY_LIMIT"
printf "\n"

# --- Validation ---
echo "✅ Verifying mirroring (requests should equal limits)..."

if [[ "$MEMORY_REQUEST" == "570Mi" && "$MEMORY_LIMIT" == "570Mi" ]]; then
  echo "✅ Memory mirroring works: request=$MEMORY_REQUEST, limit=$MEMORY_LIMIT"
else
  echo "❌ Memory mirroring failed: request='$MEMORY_REQUEST', limit='$MEMORY_LIMIT' (expected both '570Mi')"
  exit 1
fi
