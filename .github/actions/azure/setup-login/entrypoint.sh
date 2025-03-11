#!/bin/bash
set -e  # Exit immediately if a command fails
set -o pipefail  # Exit on errors in pipelines

echo "🔍 Checking Azure authentication method..."

if [[ -n "$AZURE_CLIENT_ID" && -n "$AZURE_TENANT_ID" && -n "$AZURE_SUBSCRIPTION_ID" ]]; then
  echo "🟢 Local Azure credentials detected (Running with act). Logging in with Service Principal..."

  az login --service-principal \
    --username "$AZURE_CLIENT_ID" \
    --password "$AZURE_CLIENT_SECRET" \
    --tenant "$AZURE_TENANT_ID"

  az account set --subscription "$AZURE_SUBSCRIPTION_ID"

  echo "✅ Azure authentication configured locally."

elif [[ -n "$GITHUB_ACTIONS_OIDC" ]]; then
  echo "🟡 No local credentials found. Using OIDC authentication in GitHub Actions..."

  az login --identity

  echo "✅ Azure authentication configured using OIDC."
else
  echo "❌ No valid authentication method found."
  exit 1
fi