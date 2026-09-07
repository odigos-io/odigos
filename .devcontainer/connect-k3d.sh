#!/usr/bin/env bash
set -euo pipefail

log(){ echo "[connect-k3d] $*"; }

K3D_CLUSTER="${K3D_CLUSTER:-odigos}"
K3D_NETWORK="k3d-${K3D_CLUSTER}"

log "Hooking devcontainer into k3d cluster '${K3D_CLUSTER}'..."

# Refresh kubeconfig if cluster exists
if command -v k3d >/dev/null 2>&1 && k3d cluster list 2>/dev/null | awk 'NR>1 {print $1}' | grep -qx "${K3D_CLUSTER}"; then
  k3d kubeconfig get "${K3D_CLUSTER}" > /home/vscode/.kube/config
  chown -R vscode:vscode /home/vscode/.kube
  log "Updated kubeconfig for k3d cluster '${K3D_CLUSTER}'."
else
  log "No k3d cluster '${K3D_CLUSTER}' found. Run 'k3d cluster create --config=tests/common/apply/k3d-config.yaml' first."
  exit 0
fi

# Attach container to k3d network if present
if docker network inspect "${K3D_NETWORK}" >/dev/null 2>&1; then
  hn="$(hostname || true)"
  if [ -n "${hn}" ] && docker inspect "${hn}" >/dev/null 2>&1; then
    cid="$(docker inspect -f '{{.Id}}' "${hn}")"
  else
    cid="$(docker ps --filter "name=vsc-" --format '{{.ID}}' | head -n1)"
  fi

  if [ -n "${cid}" ]; then
    log "Connecting container ${cid:0:12} to '${K3D_NETWORK}' network..."
    docker network connect "${K3D_NETWORK}" "$cid" 2>/dev/null || \
      log "Already connected or connect failed (benign)."
  else
    log "WARN: Could not determine container id; skipping network attach."
  fi
else
  log "No '${K3D_NETWORK}' network exists. Create the k3d cluster first."
fi

log "Connection script done."
