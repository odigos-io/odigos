#!/usr/bin/env bash

# Setup
TMPDIR="$(mktemp -d)"
CHARTDIRS=("helm/odigos")

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

if [[ $(git diff -- ${CHARTDIRS[*]} | wc -c) -ne 0 ]]; then
	echo "Helm chart dirty. Aborting."
	exit 1
fi

# Ignore errors because it will mostly always error locally
helm repo add odigos https://odigos-io.github.io/odigos-charts 2> /dev/null || true
git worktree add $TMPDIR gh-pages -f

# Update index with new packages
for chart in "${CHARTDIRS[@]}"
do
	echo "Updating $chart/Chart.yaml with version ${TAG#v}"
	sed -i -E 's/0.0.0/'"${TAG#v}"'/' $chart/Chart.yaml
done
helm package ${CHARTDIRS[*]} -d $TMPDIR
pushd $TMPDIR
prefix 'helm-chart-' *.tgz
helm repo index . --merge index.yaml --url https://github.com/$GITHUB_REPOSITORY/releases/download/$TAG/
git diff -G apiVersion

# The check avoids pushing the same tag twice and only pushes if there's a new entry in the index
if [[ $(git diff -G apiVersion | wc -c) -ne 0 ]]; then
	# Upload new packages
	gh release upload -R $GITHUB_REPOSITORY $TAG $TMPDIR/*.tgz || exit 1

	git add index.yaml
	git commit -m "update index with $TAG" && git push
	popd
	git fetch
else
	echo "No significant changes"
	popd
fi

# Roll back chart version changes
git checkout ${CHARTDIRS[*]}
git worktree remove $TMPDIR -f || echo " -> Failed to clean up temp worktree"
