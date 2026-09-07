#!/usr/bin/env bash
# Simulate a Karpenter-provisioned node: apply odigos.io/needs-init and verify it sticks.
set -euo pipefail

WORKER_NODE=$(kubectl get nodes -l '!node-role.kubernetes.io/control-plane' -o jsonpath='{.items[0].metadata.name}')
if [[ -z "${WORKER_NODE}" ]]; then
  echo "❌ could not find a worker node"
  kubectl get nodes -o wide || true
  exit 1
fi

echo "Applying startup taint to ${WORKER_NODE}"
kubectl taint node "${WORKER_NODE}" odigos.io/needs-init=:NoSchedule --overwrite

HAS_TAINT=$(kubectl get node "${WORKER_NODE}" -o json | jq '
  [.spec.taints // [] | .[] | select(.key == "odigos.io/needs-init" and .effect == "NoSchedule")] | length
')
if [[ "${HAS_TAINT}" != "1" ]]; then
  echo "❌ expected odigos.io/needs-init=NoSchedule on ${WORKER_NODE}"
  kubectl describe node "${WORKER_NODE}" | sed -n '/Taints:/,/Conditions:/p' || true
  exit 1
fi

echo "✅ startup taint present on ${WORKER_NODE}"
