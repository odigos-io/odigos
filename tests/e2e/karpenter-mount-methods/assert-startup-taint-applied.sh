#!/usr/bin/env bash
# Simulate a Karpenter-provisioned node: apply odigos.io/needs-init and verify it sticks.
set -euo pipefail

AGENT_NODE=$(kubectl get nodes -l node-role.kubernetes.io/agent -o jsonpath='{.items[0].metadata.name}')
if [[ -z "${AGENT_NODE}" ]]; then
  echo "❌ could not find a k3d/k3s agent node"
  kubectl get nodes -o wide || true
  exit 1
fi

echo "Applying startup taint to ${AGENT_NODE}"
kubectl taint node "${AGENT_NODE}" odigos.io/needs-init=:NoSchedule --overwrite

HAS_TAINT=$(kubectl get node "${AGENT_NODE}" -o json | jq '
  [.spec.taints // [] | .[] | select(.key == "odigos.io/needs-init" and .effect == "NoSchedule")] | length
')
if [[ "${HAS_TAINT}" != "1" ]]; then
  echo "❌ expected odigos.io/needs-init=NoSchedule on ${AGENT_NODE}"
  kubectl describe node "${AGENT_NODE}" | sed -n '/Taints:/,/Conditions:/p' || true
  exit 1
fi

echo "✅ startup taint present on ${AGENT_NODE}"
