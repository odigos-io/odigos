#!/usr/bin/env bash

set -e

# This script sets up the release candidate subdirectory in the gh-pages branch
# It creates the rc/ subdirectory with the necessary structure

echo "Setting up release candidate subdirectory in gh-pages..."

# Create a temporary worktree for gh-pages
TMPDIR="$(mktemp -d)"
git worktree add $TMPDIR gh-pages -f

pushd $TMPDIR

# Create rc subdirectory if it doesn't exist
mkdir -p rc

# Create index.yaml in rc subdirectory if it doesn't exist
if [ ! -f rc/index.yaml ]; then
    echo "Creating initial index.yaml for release candidates..."
    cat > rc/index.yaml << EOF
apiVersion: v1
entries: {}
generated: "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
EOF
fi

# Create README for the RC subdirectory
cat > rc/README.md << EOF
# Odigos Release Candidate Helm Charts

This directory contains release candidate versions of Odigos Helm charts.

## Usage

To add this repository for release candidates:

\`\`\`bash
helm repo add odigos-rc https://odigos-io.github.io/odigos/rc/
helm repo update
\`\`\`

## Installing Release Candidates

\`\`\`bash
helm install odigos odigos-rc/odigos --version <rc-version>
\`\`\`

## Warning

⚠️ **Release candidates are pre-release versions and may contain bugs or incomplete features.**
⚠️ **Do not use in production environments.**

For stable releases, use the main repository:
\`\`\`bash
helm repo add odigos https://odigos-io.github.io/odigos/
\`\`\`
EOF

# Commit and push
git add rc/
git commit -m "Initial setup of release candidate subdirectory" || echo "No changes to commit"
git push origin gh-pages

popd

# Clean up
git worktree remove $TMPDIR -f || echo " -> Failed to clean up temp worktree"

echo "Release candidate subdirectory setup complete!"
echo "Users can now add the RC repository with: helm repo add odigos-rc https://odigos-io.github.io/odigos/rc/" 