#!/usr/bin/env bash
# Install KWOK + Karpenter KWOK provider into the current kube context.
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

if [[ -n "${KARPENTER_DIR:-}" && -d "${KARPENTER_DIR}/kwok/charts" ]]; then
  echo "Using KARPENTER_DIR=${KARPENTER_DIR}"
else
  CACHE_DIR="${TMPDIR:-/tmp}/odigos-karpenter-main"
  if [[ ! -d "${CACHE_DIR}/kwok/charts" ]]; then
    echo "Cloning kubernetes-sigs/karpenter (main) into ${CACHE_DIR}"
    rm -rf "${CACHE_DIR}"
    git clone --depth 1 https://github.com/kubernetes-sigs/karpenter.git "${CACHE_DIR}"
  else
    echo "Reusing cached Karpenter checkout at ${CACHE_DIR}"
  fi
  KARPENTER_DIR="${CACHE_DIR}"
fi

echo "Installing KWOK..."
(
  cd "${KARPENTER_DIR}"
  ./hack/install-kwok.sh
)

echo "Building Karpenter KWOK controller into kind..."
export KWOK_REPO=kind.local
CONTROLLER_IMG=$(
  cd "${KARPENTER_DIR}"
  KO_DOCKER_REPO=kind.local ko build sigs.k8s.io/karpenter/kwok
)
IMG_REPOSITORY=$(echo "${CONTROLLER_IMG}" | cut -d ":" -f 1)
IMG_TAG=$(echo "${CONTROLLER_IMG}" | cut -d ":" -f 2 -s)
IMG_TAG="${IMG_TAG:-latest}"

echo "Applying Karpenter CRDs + chart (image ${IMG_REPOSITORY}:${IMG_TAG})..."
kubectl apply -f "${KARPENTER_DIR}/kwok/charts/crds"
helm upgrade --install karpenter "${KARPENTER_DIR}/kwok/charts" \
  --namespace kube-system \
  --create-namespace \
  --skip-crds \
  --set logLevel=debug \
  --set controller.image.repository="${IMG_REPOSITORY}" \
  --set controller.image.tag="${IMG_TAG}" \
  --set serviceMonitor.enabled=false \
  --set settings.featureGates.nodeRepair=false \
  --set settings.featureGates.spotToSpotConsolidation=false

kubectl -n kube-system set env deployment/karpenter \
  FEATURE_GATES="NodeOverlay=true,ReservedCapacity=true,SpotToSpotConsolidation=false,NodeRepair=false,StaticCapacity=false,CapacityBuffer=false"

kubectl -n kube-system rollout status deployment/karpenter --timeout=3m

echo "Applying NodePool / KWOKNodeClass (NodeOverlay is applied later by the test)..."
kubectl apply -f "${SCRIPT_DIR}/nodepool.yaml"

echo "Karpenter KWOK setup complete."
