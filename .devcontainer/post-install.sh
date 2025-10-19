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

# Check for helm, if not present, install it
if ! command -v helm &> /dev/null; then
  log "Helm not found, installing..."
  # Installing helm at tag 3.19.0
  curl -s https://raw.githubusercontent.com/helm/helm/3a5805ea7e8aa385cce278e0026656baa68fa83d/scripts/get-helm-3 | bash
  log "Helm installed."
else
  log "Helm found"
fi
# Permissions
chown -R vscode:vscode /home/vscode/.kube

# Call connection script (it will no-op gracefully if no cluster yet)
.devcontainer/connect-kind.sh || true

log "Essentials setup done."
