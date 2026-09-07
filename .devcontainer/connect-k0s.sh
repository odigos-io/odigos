#!/usr/bin/env bash
set -euo pipefail

log(){ echo "[connect-k0s] $*"; }

K0S_CONTROLLER="${K0S_CONTROLLER:-k0s-controller}"

log "Hooking devcontainer into k0s cluster..."

if ! command -v docker >/dev/null 2>&1; then
  log "docker not found; skipping"
  exit 0
fi

if ! docker inspect "${K0S_CONTROLLER}" >/dev/null 2>&1; then
  log "No k0s controller '${K0S_CONTROLLER}' found. Run './scripts/k0s-cluster.sh create' first."
  exit 0
fi

mkdir -p /home/vscode/.kube
# Refresh kubeconfig via the repo helper if present, otherwise inline.
if [[ -x /workspaces/odigos/scripts/k0s-cluster.sh ]]; then
  KUBECONFIG=/home/vscode/.kube/config /workspaces/odigos/scripts/k0s-cluster.sh kubeconfig
else
  docker exec "${K0S_CONTROLLER}" k0s kubeconfig admin > /home/vscode/.kube/config
  sed -i -E 's|https://[^:]+:6443|https://127.0.0.1:6443|' /home/vscode/.kube/config || true
fi
chown -R vscode:vscode /home/vscode/.kube
log "Updated kubeconfig for k0s."

# Attach container to the default bridge if needed is a no-op for k0s (uses Docker bridge DNS carefully).
# Ensure the controller is reachable on published 6443 from this container via host gateway.
log "Connection script done."
