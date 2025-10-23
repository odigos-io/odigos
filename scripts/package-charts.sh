#!/usr/bin/env bash

set -e

# Setup
CHARTDIRS=("helm/odigos" "helm/odigos-central")

if [ -z "$TAG" ]; then
	echo "TAG required"
	exit 1
fi

if [[ $(git diff -- ${CHARTDIRS[*]} | wc -c) -ne 0 ]]; then
	echo "Helm chart dirty. Aborting."
	exit 1
fi

echo "------------------------------------------------------------"
echo "📦 Packaging Helm charts for version ${TAG#v}"
echo "------------------------------------------------------------"

# Update chart versions and package them
for chart in "${CHARTDIRS[@]}"
do
	echo "Updating $chart/Chart.yaml with version ${TAG#v}"
	sed -i -E 's/0.0.0/'"${TAG#v}"'/' $chart/Chart.yaml
done

# Package charts to helm/ directory
helm package ${CHARTDIRS[*]} -d helm/

echo "✅ Helm charts packaged successfully:"
ls -lah helm/odigos-*.tgz

# Roll back chart version changes
git checkout ${CHARTDIRS[*]}

echo "✅ Chart packaging completed"
