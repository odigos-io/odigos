#!/usr/bin/env bash

set -e

# Setup
TMPDIR="$(mktemp -d)"
CHARTDIRS=("helm/odigos" "helm/odigos-central")

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

git worktree add $TMPDIR gh-pages -f

# Update index with new packages
for chart in "${CHARTDIRS[@]}"
do
	echo "Updating $chart/Chart.yaml with version ${TAG#v}"
	sed -i -E 's/0.0.0/'"${TAG#v}"'/' $chart/Chart.yaml
done
helm package ${CHARTDIRS[*]} -d $TMPDIR
cp $TMPDIR/odigos-*.tgz helm/
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
  echo "🔎 Checking if release $TAG exists in $GITHUB_REPOSITORY..."
  gh release view -R "$GITHUB_REPOSITORY" "$TAG" || echo "⚠️ Release not found, will attempt to create it"
  echo "------------------------------------------------------------"

  if ! gh release view -R "$GITHUB_REPOSITORY" "$TAG" > /dev/null 2>&1; then
    echo "🚀 Creating GitHub release $TAG..."
    set -x
    if ! gh release create -R "$GITHUB_REPOSITORY" "$TAG" --title "$TAG" --notes "Auto-created for Helm charts"; then
      echo "❌ Failed to create release $TAG"
      exit 1
    fi
    set +x
    echo "✅ Release $TAG created successfully"
  else
    echo "✅ Release already exists, continuing"
  fi

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


# Roll back chart version changes
git checkout ${CHARTDIRS[*]}
git worktree remove $TMPDIR -f || echo " -> Failed to clean up temp worktree"
