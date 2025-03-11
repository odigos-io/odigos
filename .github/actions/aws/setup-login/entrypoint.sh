#!/bin/bash
set -e  # Exit immediately if a command fails
set -o pipefail  # Exit on errors in pipelines

echo "🔍 Checking AWS authentication method..."

if [[ -n "$AWS_ACCESS_KEY_ID" ]]; then
  echo "🟢 Local AWS credentials detected (Running with act). Configuring ~/.aws/credentials..."

  mkdir -p ~/.aws
  cat <<EOF > ~/.aws/credentials
[default]
aws_access_key_id=$AWS_ACCESS_KEY_ID
aws_secret_access_key=$AWS_SECRET_ACCESS_KEY
aws_session_token=$AWS_SESSION_TOKEN
EOF

  echo "✅ AWS credentials configured locally."

else
  echo "🟡 No local credentials found. Using OIDC authentication in GitHub Actions..."

  aws-actions/configure-aws-credentials@v4 \
    --role-to-assume "$AWS_ROLE_TO_ASSUME" \
    --aws-region "$AWS_REGION"

  echo "✅ AWS credentials configured using OIDC."
fi