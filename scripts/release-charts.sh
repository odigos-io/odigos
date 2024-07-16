#!/usr/bin/env sh

# Setup
TMPDIR="$(mktemp -d)"

if [ -z "$TAG" ]; then
	echo "TAG required"
	exit 1
fi

if [ -z "$GITHUB_REPOSITORY" ]; then
	echo "GITHUB_REPOSITORY required"
	exit 1
fi

helm repo add odigos https://odigos-io.github.io/odigos-charts || true
git worktree add $TMPDIR gh-pages -f

# Update index with new packages
helm package helm/* -d $TMPDIR
cd $TMPDIR
helm repo index . --merge index.yaml --url https://github.com/$GITHUB_REPOSITORY/releases/download/$TAG/

if [[ $(git diff -G apiVersion | wc -c) -ne 0 ]]; then
	# Upload new packages
	gh release upload -R $GITHUB_REPOSITORY $TAG $TMPDIR/*.tgz

	git add index.yaml
	git commit -m "update index" && git push
else
	echo "No significant changes"
fi

git worktree remove $TMPDIR || echo " -> Failed to clean up temp worktree"
