#!/usr/bin/env bash
# Wait until Karpenter creates at least one NodeClaim.
set -euo pipefail

DEADLINE=$((SECONDS + 180))
while (( SECONDS < DEADLINE )); do
  COUNT=$(kubectl get nodeclaims -o json 2>/dev/null | jq '.items | length')
  if [[ "${COUNT}" -ge 1 ]]; then
    echo "✅ NodeClaim created (count=${COUNT})"
    kubectl get nodeclaims -o wide || true
    exit 0
  fi
  echo "waiting for NodeClaim... (${SECONDS}s)"
  sleep 2
done

echo "❌ timed out waiting for NodeClaim"
kubectl get pods -n default -l app=inventory -o wide || true
kubectl get nodeclaims -o yaml || true
kubectl logs -n kube-system deploy/karpenter --tail=80 || true
exit 1
