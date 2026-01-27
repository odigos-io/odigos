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

# Install kubectl v1.34.1
log "Installing kubectl v1.34.1..."
ARCH=$(dpkg --print-architecture) && \
    curl -LO "https://dl.k8s.io/release/v1.34.1/bin/linux/${ARCH}/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
    rm kubectl

# Install kind (latest stable release)
log "Installing kind..."
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.23.0/kind-linux-$(dpkg --print-architecture) && \
    install -o root -g root -m 0755 kind /usr/local/bin/kind && \
    rm kind

# Add alias kc=kubectl + enable completion
log "Adding kubectl alias and completion..."
echo "alias kc='kubectl --insecure-skip-tls-verify=true'" >> /etc/bash.bashrc \
    && echo "source <(kubectl completion bash)" >> /etc/bash.bashrc

# Install ko (latest release) for testing
log "Installing ko..."
KO_VERSION=$(curl -s https://api.github.com/repos/ko-build/ko/releases/latest \
    | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/') && \
    curl -L https://github.com/ko-build/ko/releases/download/v${KO_VERSION}/ko_Linux_$(dpkg --print-architecture).tar.gz \
    | tar xz -C /usr/local/bin

log "Install Chainsaw..."
go install github.com/kyverno/chainsaw@latest

# Permissions
log "Setting permissions..."
chown -R vscode:vscode /home/vscode/.kube

# Call connection script (it will no-op gracefully if no cluster yet)
log "Attempting to connect to kind cluster..."
.devcontainer/connect-kind.sh || true
.devcontainer/init-docs.sh || true

log "Essentials setup done."
