#!/usr/bin/env bash

set -e

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

# Function to detect if this is a release candidate
is_release_candidate() {
	local tag="$1"
	# Check if tag contains rc, alpha, beta, or pre-release indicators
	if [[ "$tag" =~ -(rc) ]]; then
		return 0  # true
	else
		return 1  # false
	fi
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

# Determine if this is a release candidate
if is_release_candidate "$TAG"; then
	echo "Detected release candidate: $TAG"
	RC_SUBDIR="rc"
	COMMIT_MSG="update rc index with $TAG"
	HELM_REPO_URL="https://github.com/$GITHUB_REPOSITORY/releases/download/$TAG/"
else
	echo "Detected stable release: $TAG"
	RC_SUBDIR=""
	COMMIT_MSG="update index with $TAG"
	HELM_REPO_URL="https://github.com/$GITHUB_REPOSITORY/releases/download/$TAG/"
fi

# Create worktree for gh-pages branch
git worktree add $TMPDIR gh-pages -f

# Update index with new packages
for chart in "${CHARTDIRS[@]}"
do
	echo "Updating $chart/Chart.yaml with version ${TAG#v}"
	sed -i -E 's/0.0.0/'"${TAG#v}"'/' $chart/Chart.yaml
done
helm package ${CHARTDIRS[*]} -d $TMPDIR
pushd $TMPDIR

# For release candidates, create and work in the rc subdirectory
if [ -n "$RC_SUBDIR" ]; then
	mkdir -p "$RC_SUBDIR"
	mv *.tgz "$RC_SUBDIR/"
	cd "$RC_SUBDIR"
fi

prefix 'helm-chart-' *.tgz

# Use different index files for RC vs stable
if [ -n "$RC_SUBDIR" ]; then
	helm repo index . --merge index.yaml --url "$HELM_REPO_URL"
else
	helm repo index . --merge index.yaml --url "$HELM_REPO_URL"
fi
git diff -G apiVersion

# The check avoids pushing the same tag twice and only pushes if there's a new entry in the index
if [[ $(git diff -G apiVersion | wc -c) -ne 0 ]]; then
	# Upload new packages
	if [ -n "$RC_SUBDIR" ]; then
		gh release upload -R $GITHUB_REPOSITORY $TAG $TMPDIR/$RC_SUBDIR/*.tgz || exit 1
	else
		gh release upload -R $GITHUB_REPOSITORY $TAG $TMPDIR/*.tgz || exit 1
	fi

	# Add the appropriate index file
	if [ -n "$RC_SUBDIR" ]; then
		git add "$RC_SUBDIR/index.yaml"
	else
		git add index.yaml
	fi
	
	git commit -m "$COMMIT_MSG" && git push
	popd
	git fetch
else
	echo "No significant changes"
	popd
fi

# Roll back chart version changes
git checkout ${CHARTDIRS[*]}
git worktree remove $TMPDIR -f || echo " -> Failed to clean up temp worktree"
