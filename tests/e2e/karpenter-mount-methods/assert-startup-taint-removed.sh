#!/usr/bin/env bash
# Rollout odiglet so init runs PrepareNodeForOdigosInstallation and clears odigos.io/needs-init.
set -euo pipefail

echo "Restarting odiglet so init removes odigos.io/needs-init from kind-worker"
kubectl rollout restart daemonset/odiglet -n odigos-test
kubectl rollout status daemonset/odiglet -n odigos-test --timeout=5m

DEADLINE=$((SECONDS + 180))
while (( SECONDS < DEADLINE )); do
  HAS_TAINT=$(kubectl get node kind-worker -o json | jq '
    [.spec.taints // [] | .[] | select(.key == "odigos.io/needs-init" and .effect == "NoSchedule")] | length
  ')
  if [[ "${HAS_TAINT}" == "0" ]]; then
    echo "✅ startup taint removed from kind-worker"
    exit 0
  fi
  echo "waiting for odiglet to remove odigos.io/needs-init from kind-worker..."
  sleep 2
done

echo "❌ timed out waiting for taint removal on kind-worker"
kubectl describe node kind-worker | sed -n '/Taints:/,/Conditions:/p' || true
kubectl -n odigos-test logs daemonset/odiglet --tail=80 || true
exit 1
