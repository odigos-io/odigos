#!/usr/bin/env bash
# Simulate a Karpenter-provisioned node: apply odigos.io/needs-init and verify it sticks.
set -euo pipefail

echo "Applying startup taint to kind-worker"
kubectl taint node kind-worker odigos.io/needs-init=:NoSchedule --overwrite

HAS_TAINT=$(kubectl get node kind-worker -o json | jq '
  [.spec.taints // [] | .[] | select(.key == "odigos.io/needs-init" and .effect == "NoSchedule")] | length
')
if [[ "${HAS_TAINT}" != "1" ]]; then
  echo "❌ expected odigos.io/needs-init=NoSchedule on kind-worker"
  kubectl describe node kind-worker | sed -n '/Taints:/,/Conditions:/p' || true
  exit 1
fi

echo "✅ startup taint present on kind-worker"
