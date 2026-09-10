#!/usr/bin/env bash
set -euo pipefail

# Kubernetes rejects "Bidirectional" mount propagation on a container that is not privileged
# ("Bidirectional mount propagation is available only to privileged containers"), so a render
# that violates this is accepted by `helm lint` but rejected by the API server on apply.
# This renders the odiglet DaemonSet for the value combinations that add such a mount and
# asserts every container carrying one is privileged.

# Accept optional path argument
P=${1:-"../../.."}

check_render() {
  local description=$1
  shift

  echo "🚀 Rendering odiglet DaemonSet: $description"
  helm template odigos "$P/helm/odigos" \
    --show-only templates/odiglet/daemonset.yaml \
    "$@" > /tmp/odiglet-privileged-mounts.yaml

  local offenders
  offenders=$(yq '
    [.spec.template.spec.initContainers[]?, .spec.template.spec.containers[]?]
    | .[]
    | select([.volumeMounts[]? | select(.mountPropagation == "Bidirectional")] | length > 0)
    | select(.securityContext.privileged != true)
    | .name' /tmp/odiglet-privileged-mounts.yaml)

  if [ -n "$offenders" ]; then
    echo "❌ containers with Bidirectional mount propagation that are not privileged:"
    echo "$offenders"
    return 1
  fi

  echo "✅ every container with Bidirectional mount propagation is privileged"
  echo
}

check_render "defaults"
check_render "openshift.enabled=true" --set openshift.enabled=true
check_render "openshift.enabled=true, mountMethod=k8s-csi-driver" \
  --set openshift.enabled=true --set instrumentor.mountMethod=k8s-csi-driver

echo "✅ odiglet privileged mount propagation checks passed"
