#!/usr/bin/env bash
set -euo pipefail

log(){ echo "[post-install] $*"; }

log "Running essentials setup..."

# Ensure ~/.kube exists
mkdir -p /home/vscode/.kube

# Copy host kubeconfig if present
if [ -f /home/vscode/.kube-host/config ]; then
  cp /home/vscode/.kube-host/config /home/vscode/.kube/config
  log "Copied host kubeconfig."
fi

# Permissions
chown -R vscode:vscode /home/vscode/.kube

# Call connection script (it will no-op gracefully if no cluster yet)
.devcontainer/connect-kind.sh || true

log "Essentials setup done."
