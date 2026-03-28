#!/usr/bin/env bash
# Push Odigos images built as ${SOURCE_ORG}/odigos-<component>:${TAG} to AWS Public ECR
# using the Helm imagePrefix layout:
#   public.ecr.aws/<alias>/<prefix>/odigos-<component>:<tag>
#
# Default prefix matches: imagePrefix=public.ecr.aws/odigos/odigos/core/profiler
#
# Prerequisites:
#   - aws CLI credentials with ecr-public:BatchCheckLayerAvailability, PutImage, InitiateLayerUpload, etc.
#   - docker login to public.ecr.aws (this script runs ecr-public get-login-password)
#   - Images already built only for the components you need, e.g. profiling delta:
#       TAG=$(git rev-parse HEAD)
#       make build-collector build-odiglet build-autoscaler build-ui TAG="$TAG" ORG=registry.odigos.io
#   Full suite: COMPONENTS=all and make build-images + build-cli-image (same TAG).
#
# COMPONENTS: space- or comma-separated list, or "all". Default is the profiling-related set.
#
set -euo pipefail

TAG="${TAG:-$(git -C "$(dirname "$0")/.." rev-parse HEAD)}"
SOURCE_ORG="${SOURCE_ORG:-registry.odigos.io}"
# Registry alias + path prefix (no trailing slash). Must match full refs in Helm values `images`.
IMAGE_PREFIX="${IMAGE_PREFIX:-public.ecr.aws/odigos/odigos/core/profiler}"

_raw="${COMPONENTS:-}"
if [[ "$_raw" == "all" ]]; then
  read -ra COMPONENTS_ARR <<< "collector instrumentor ui scheduler autoscaler odiglet agents cli"
elif [[ -n "$_raw" ]]; then
  _raw="${_raw//,/ }"
  read -ra COMPONENTS_ARR <<< "$_raw"
else
  read -ra COMPONENTS_ARR <<< "collector odiglet autoscaler ui"
fi

echo "Logging in to public.ecr.aws ..."
aws ecr-public get-login-password --region us-east-1 \
  | docker login --username AWS --password-stdin public.ecr.aws

for c in "${COMPONENTS_ARR[@]}"; do
  src="${SOURCE_ORG}/odigos-${c}:${TAG}"
  dst="${IMAGE_PREFIX}/odigos-${c}:${TAG}"
  echo "Tag + push ${dst}"
  docker tag "${src}" "${dst}"
  docker push "${dst}"
done

echo "Done. Hybrid Helm: set image.tag to stable chart version; list only pushed images under values.images (see helm/odigos/values.profiler-dev.images.yaml)."
