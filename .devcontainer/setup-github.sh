#!/usr/bin/env bash
set -euo pipefail

log(){ echo "[setup-github] $*"; }

ENV_FILE="/workspaces/odigos/.env"

if [[ -f "$ENV_FILE" ]]; then
  log "Loading environment from $ENV_FILE"
  # Read non-empty, non-comment lines and export safely
  while IFS='=' read -r key value; do
    # skip comments and empty lines
    [[ -z "$key" || "$key" =~ ^# ]] && continue
    export "$key"="$value"
  done < "$ENV_FILE"
fi

# Ensure GOPRIVATE is set to include the private repo pattern; set to the rc files
for rcfile in /home/vscode/.bashrc /root/.bashrc; do
  if ! grep -q "GOPRIVATE" "$rcfile"; then
    echo 'export GOPRIVATE=github.com/odigos-io/*' >> "$rcfile"
    log "Appended GOPRIVATE to $rcfile"
  fi
done


if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  log "ERROR: GITHUB_TOKEN not set. Private Go modules will fail."
  exit 1
fi

if [[ -n "${GITHUB_TOKEN:-}" ]]; then
  mkdir -p /run/secrets
  echo -n "$GITHUB_TOKEN" > /run/secrets/github_token
  chmod 600 /run/secrets/github_token
  log "Wrote GITHUB_TOKEN to /run/secrets/github_token (for docker --secret)."
fi

# Configure git
git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"
log "Configured git with GITHUB_TOKEN (first 5 chars: ${GITHUB_TOKEN:0:5}****)"

# Persist for future shells
for rcfile in /home/vscode/.bashrc /root/.bashrc; do
  if ! grep -q "GITHUB_TOKEN" "$rcfile"; then
    echo "export GITHUB_TOKEN=${GITHUB_TOKEN}" >> "$rcfile"
    log "Appended GITHUB_TOKEN to $rcfile"
  fi
done
