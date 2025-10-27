#!/usr/bin/env bash
set -euo pipefail

log(){ echo "[connect-kind] $*"; }

log "Hooking devcontainer into kind..."

# Refresh kubeconfig if cluster exists
if command -v kind >/dev/null 2>&1 && kind get clusters | grep -qx kind; then
  kind get kubeconfig --name kind --internal > /home/vscode/.kube/config
  chown -R vscode:vscode /home/vscode/.kube
  log "Updated kubeconfig for kind."
else
  log "No kind cluster found. Run 'kind create cluster' first."
  exit 0
fi

# Attach container to kind network if present
if docker network inspect kind >/dev/null 2>&1; then
  hn="$(hostname || true)"
  if [ -n "${hn}" ] && docker inspect "${hn}" >/dev/null 2>&1; then
    cid="$(docker inspect -f '{{.Id}}' "${hn}")"
  else
    cid="$(docker ps --filter "name=vsc-" --format '{{.ID}}' | head -n1)"
  fi

  if [ -n "${cid}" ]; then
    log "Connecting container ${cid:0:12} to 'kind' network..."
    docker network connect kind "$cid" 2>/dev/null || \
      log "Already connected or connect failed (benign)."
  else
    log "WARN: Could not determine container id; skipping network attach."
  fi
else
  log "No 'kind' network exists. Run 'kind create cluster' first."
fi

log "Connection script done."
