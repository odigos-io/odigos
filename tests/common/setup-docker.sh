#!/bin/bash
set -euo pipefail

# Script to set up Docker for Odigos e2e testing on Ubuntu
# This script handles Docker installation and permission configuration

log() {
    echo "[setup-docker] $*"
}

# Check if running as root
if [ "$EUID" -eq 0 ]; then 
    log "Please do not run this script as root. It will use sudo when needed."
    exit 1
fi

# Detect if Docker is already installed
if command -v docker &> /dev/null; then
    log "Docker is already installed"
    if docker info &> /dev/null; then
        log "Docker daemon is accessible. Setup complete."
        exit 0
    else
        log "Docker is installed but daemon is not accessible. Fixing permissions..."
    fi
else
    log "Installing Docker..."
    
    # Update package index
    sudo apt-get update
    
    # Install prerequisites
    sudo apt-get install -y \
        ca-certificates \
        curl \
        gnupg \
        lsb-release
    
    # Add Docker's official GPG key
    sudo install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    sudo chmod a+r /etc/apt/keyrings/docker.gpg
    
    # Set up Docker repository
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
      $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
    
    # Install Docker packages with --allow-change-held-packages flag
    # This is needed because containerd.io might be held/pinned by the system
    sudo apt-get update
    sudo apt-get install -y --allow-change-held-packages \
        docker-ce \
        docker-ce-cli \
        containerd.io \
        docker-buildx-plugin \
        docker-compose-plugin
    
    log "Docker installed successfully"
fi

# Add current user to docker group if not already added
if ! groups | grep -q docker; then
    log "Adding user to docker group..."
    sudo usermod -aG docker "$USER"
    log "User added to docker group. You may need to log out and back in for this to take effect."
    log "Alternatively, you can run: newgrp docker"
    
    # Note: User needs to log out/in or run 'newgrp docker' for group changes to take effect
    log "To activate docker group without logging out, run: newgrp docker"
else
    log "User is already in docker group"
fi

# Start and enable Docker service
log "Starting Docker service..."
sudo systemctl start docker
sudo systemctl enable docker

# Verify Docker is working
log "Verifying Docker installation..."
if docker info &> /dev/null; then
    log "✅ Docker is working correctly!"
    docker --version
else
    log "❌ Docker daemon is not accessible."
    log "Please try:"
    log "  1. Log out and log back in"
    log "  2. Or run: newgrp docker"
    log "  3. Or run: sudo systemctl restart docker"
    exit 1
fi

log "Docker setup complete!"

