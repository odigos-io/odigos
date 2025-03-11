#!/bin/bash
set -e  # Exit immediately if a command fails
set -o pipefail  # Exit on errors in pipelines

echo "🔍 Checking for AWS CLI..."

# Check if AWS CLI is installed
if command -v aws &> /dev/null; then
    echo "✅ AWS CLI is already installed. Version:"
    aws --version
else
    echo "⚠️ AWS CLI not found. Installing..."

    # Detect OS and Architecture
    ARCH=$(uname -m)
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        PLATFORM="linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        PLATFORM="mac"
    else
        echo "❌ Unsupported OS: $OSTYPE"
        exit 1
    fi

    # Download and install AWS CLI
    curl "https://awscli.amazonaws.com/awscli-exe-${PLATFORM}-${ARCH}.zip" -o "awscliv2.zip"
    unzip awscliv2.zip
    sudo ./aws/install

    # Verify installation
    if command -v aws &> /dev/null; then
        echo "✅ AWS CLI installed successfully."
        aws --version
    else
        echo "❌ AWS CLI installation failed."
        exit 1
    fi
fi