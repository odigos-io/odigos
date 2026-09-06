#!/usr/bin/env bash
# Rollout odiglet so init runs PrepareNodeForOdigosInstallation and clears odigos.io/needs-init.
set -euo pipefail

WORKER_NODE=$(kubectl get nodes -l '!node-role.kubernetes.io/control-plane' -o jsonpath='{.items[0].metadata.name}')
if [[ -z "${WORKER_NODE}" ]]; then
  echo "❌ could not find a worker node"
  kubectl get nodes -o wide || true
  exit 1
fi

echo "Restarting odiglet so init removes odigos.io/needs-init from ${WORKER_NODE}"
kubectl rollout restart daemonset/odiglet -n odigos-test
kubectl rollout status daemonset/odiglet -n odigos-test --timeout=5m

DEADLINE=$((SECONDS + 180))
while (( SECONDS < DEADLINE )); do
  HAS_TAINT=$(kubectl get node "${WORKER_NODE}" -o json | jq '
    [.spec.taints // [] | .[] | select(.key == "odigos.io/needs-init" and .effect == "NoSchedule")] | length
  ')
  if [[ "${HAS_TAINT}" == "0" ]]; then
    echo "✅ startup taint removed from ${WORKER_NODE}"
    exit 0
  fi
  echo "waiting for odiglet to remove odigos.io/needs-init from ${WORKER_NODE}..."
  sleep 2
done

echo "❌ timed out waiting for taint removal on ${WORKER_NODE}"
kubectl describe node "${WORKER_NODE}" | sed -n '/Taints:/,/Conditions:/p' || true
kubectl -n odigos-test logs daemonset/odiglet --tail=80 || true
exit 1
