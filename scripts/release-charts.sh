#!/usr/bin/env sh

# Setup
TMPDIR="$(mktemp -d)"
CHARTDIR="helm/odigos"

if [ -z "$TAG" ]; then
	echo "TAG required"
	exit 1
fi

if [ -z "$GITHUB_REPOSITORY" ]; then
	echo "GITHUB_REPOSITORY required"
	exit 1
fi

if [[ $(git diff -- $CHARTDIR | wc -c) -ne 0 ]]; then
	echo "Helm chart dirty. Aborting."
	exit 1
fi

# Ignore errors because it will mostly always error locally
helm repo add odigos https://odigos-io.github.io/odigos-charts 2> /dev/null || true
git worktree add $TMPDIR gh-pages -f

# Update index with new packages
sed -i -E 's/v0.0.0/'"${TAG}"'/' $CHARTDIR/Chart.yaml
helm package helm/* -d $TMPDIR
pushd $TMPDIR
helm repo index . --merge index.yaml --url https://github.com/$GITHUB_REPOSITORY/releases/download/$TAG/

# The check avoids pushing the same tag twice and only pushes if there's a new entry in the index
if [[ $(git diff -G apiVersion | wc -c) -ne 0 ]]; then
	# Upload new packages
	rename 'odigos' 'helm-chart-odigos' *.tgz
	gh release upload -R $GITHUB_REPOSITORY $TAG $TMPDIR/*.tgz

	git add index.yaml
	git commit -m "update index with $TAG" && git push
	popd
	git fetch
else
	echo "No significant changes"
	popd
fi

# Roll back chart version changes
git checkout $CHARTDIR
git worktree remove $TMPDIR -f || echo " -> Failed to clean up temp worktree"
