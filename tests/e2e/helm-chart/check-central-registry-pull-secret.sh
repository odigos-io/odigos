#!/usr/bin/env bash
set -euo pipefail

# Accept optional path argument
P=${1:-"../../.."}

DEFAULT_REGISTRY_RENDER="${TMPDIR:-/tmp}/odigos-central-default-registry.yaml"
CUSTOM_REGISTRY_RENDER="${TMPDIR:-/tmp}/odigos-central-custom-registry.yaml"

echo "Rendering odigos-central with explicit default registry..."
helm template odigos-central "$P/helm/odigos-central" \
  --namespace odigos-central \
  --set onPremToken=test-token \
  --set imagePrefix=registry.odigos.io > "$DEFAULT_REGISTRY_RENDER"

if ! grep -q "image: registry.odigos.io/odigos-enterprise-central-backend:" "$DEFAULT_REGISTRY_RENDER"; then
  echo "Expected central-backend to render from registry.odigos.io"
  exit 1
fi

if ! grep -q "name: odigos-enterprise-registry" "$DEFAULT_REGISTRY_RENDER"; then
  echo "Expected odigos-central to create odigos-enterprise-registry for registry.odigos.io images"
  exit 1
fi

PULL_SECRET_MOUNTS=$(grep -c 'name: "odigos-enterprise-registry"' "$DEFAULT_REGISTRY_RENDER" || true)
if [[ "$PULL_SECRET_MOUNTS" -lt 2 ]]; then
  echo "Expected central workloads to mount odigos-enterprise-registry, found $PULL_SECRET_MOUNTS mounts"
  exit 1
fi

echo "Rendering odigos-central with a custom registry..."
helm template odigos-central "$P/helm/odigos-central" \
  --namespace odigos-central \
  --set onPremToken=test-token \
  --set imagePrefix=mirror.example.com/odigos > "$CUSTOM_REGISTRY_RENDER"

if grep -q "name: odigos-enterprise-registry" "$CUSTOM_REGISTRY_RENDER"; then
  echo "Did not expect odigos-enterprise-registry when images come from a custom registry"
  exit 1
fi

echo "Central registry pull secret rendering works as expected."
