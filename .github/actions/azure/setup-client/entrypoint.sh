#!/bin/bash
set -e  # Exit immediately if a command fails
set -o pipefail  # Exit on errors in pipelines

echo "🔍 Checking for Azure CLI..."

# Check if Azure CLI is installed
if command -v az &> /dev/null; then
    echo "✅ Azure CLI is already installed. Version:"
    az --version
else
    echo "⚠️ Azure CLI not found. Installing..."

    # Detect OS
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        PLATFORM="linux"
        curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        PLATFORM="mac"
        brew install azure-cli
    else
        echo "❌ Unsupported OS: $OSTYPE"
        exit 1
    fi

    # Verify installation
    if command -v az &> /dev/null; then
        echo "✅ Azure CLI installed successfully."
        az --version
    else
        echo "❌ Azure CLI installation failed."
        exit 1
    fi
fi