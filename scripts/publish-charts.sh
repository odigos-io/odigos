#!/usr/bin/env bash

set -e

# Setup
TMPDIR="$(mktemp -d)"

prefix () {
	echo "${@:1}"
	echo "${@:2}"
	for i in "${@:2}"; do
		echo "Renaming $i to $1$i"
		mv "$i" "$1$i"
	done
}

if [ -z "$TAG" ]; then
	echo "TAG required"
	exit 1
fi

if [ -z "$GITHUB_REPOSITORY" ]; then
	echo "GITHUB_REPOSITORY required"
	exit 1
fi

echo "------------------------------------------------------------"
echo "📦 Publishing pre-packaged Helm charts for $TAG"
echo "------------------------------------------------------------"

# Verify that the packaged charts exist
if [ ! -f "helm/odigos-${TAG#v}.tgz" ] || [ ! -f "helm/odigos-central-${TAG#v}.tgz" ]; then
	echo "❌ Pre-packaged charts not found in helm/ directory"
	echo "Expected: helm/odigos-${TAG#v}.tgz and helm/odigos-central-${TAG#v}.tgz"
	ls -la helm/
	exit 1
fi

echo "✅ Found pre-packaged charts:"
ls -lah helm/odigos-*.tgz

git worktree add $TMPDIR gh-pages -f

# Copy the pre-packaged charts to temp directory
cp helm/odigos-*.tgz $TMPDIR/
pushd $TMPDIR
prefix 'helm-chart-' *.tgz
helm repo index . --merge index.yaml --url https://github.com/$GITHUB_REPOSITORY/releases/download/$TAG/
git diff -G apiVersion

# The check avoids pushing the same tag twice and only pushes if there's a new entry in the index
if [[ $(git diff -G apiVersion | wc -c) -ne 0 ]]; then
  echo "------------------------------------------------------------"
  echo "🔍 Debug info before uploading Helm charts"
  echo "TAG: $TAG"
  echo "GITHUB_REPOSITORY: $GITHUB_REPOSITORY"
  echo "Current working dir: $(pwd)"
  echo "Files in TMPDIR:"
  ls -lah "$TMPDIR"
  echo "------------------------------------------------------------"
  echo "🔐 Checking GitHub CLI authentication status:"
  gh auth status || echo "⚠️ gh auth status failed"
  echo "------------------------------------------------------------"
  echo "🔎 Verifying release $TAG exists (should be created by GoReleaser)..."
  if ! gh release view -R "$GITHUB_REPOSITORY" "$TAG" > /dev/null 2>&1; then
    echo "❌ Release $TAG not found. GoReleaser should have created it."
    exit 1
  fi
  echo "✅ Release $TAG exists, proceeding with upload"
  echo "------------------------------------------------------------"

  echo "📦 Uploading Helm chart packages to release $TAG..."
  set -x
  if ! gh release upload -R "$GITHUB_REPOSITORY" "$TAG" "$TMPDIR"/*.tgz; then
    echo "❌ Failed to upload Helm charts to release $TAG"
    exit 1
  fi
  set +x
  echo "✅ Upload completed successfully"
  echo "------------------------------------------------------------"

  git add index.yaml
  git commit -m "update index with $TAG" && git push
  popd
  git fetch
else
  echo "No significant changes"
  popd
fi

# Clean up temp worktree
git worktree remove $TMPDIR -f || echo " -> Failed to clean up temp worktree"

echo "✅ Helm charts published successfully"
