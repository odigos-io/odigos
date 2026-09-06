#!/usr/bin/env bash
# After the namespace is already instrumented (virtual-device):
# 1) Patch inventory resources (device request) and scale to 5
# 2) Taint kind nodes so new pods cannot land there (otherwise kind has ~1024 devices and never Pendings)
# 3) Without NodeOverlay: pods stay Pending, no NodeClaim
# 4) Apply NodeOverlay: Karpenter creates a NodeClaim
# 5) Immediately delete NodeClaims + KWOK nodes — they break kindnet CNI routes.
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

delete_kwok_residue() {
  kubectl delete nodeclaims --all --ignore-not-found --wait=false >/dev/null 2>&1 || true
  kubectl delete nodes -l kwok.x-k8s.io/node --ignore-not-found --force --grace-period=0 >/dev/null 2>&1 || true
  kubectl get nodes -o name 2>/dev/null | grep '^node/kwok-' | xargs -r kubectl delete --ignore-not-found --force --grace-period=0 >/dev/null 2>&1 || true
}

repair_kindnet() {
  kindnet_pod=$(kubectl get pods -n kube-system -l app=kindnet \
    --field-selector spec.nodeName=kind-control-plane \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
  if [[ -n "${kindnet_pod}" ]]; then
    kubectl delete pod -n kube-system "${kindnet_pod}" --ignore-not-found --wait=false >/dev/null 2>&1 || true
  fi
}

taint_kind_nodes() {
  for node in $(kubectl get nodes -o jsonpath='{.items[*].metadata.name}'); do
    kubectl taint node "${node}" CriticalAddonsOnly=true:NoSchedule --overwrite >/dev/null
  done
}

untaint_kind_nodes() {
  for node in $(kubectl get nodes -o jsonpath='{.items[*].metadata.name}'); do
    kubectl taint node "${node}" CriticalAddonsOnly- >/dev/null 2>&1 || true
  done
}

cleanup() {
  kubectl delete nodeoverlay odigos-instrumentation --ignore-not-found >/dev/null 2>&1 || true
  delete_kwok_residue
  repair_kindnet
  untaint_kind_nodes
  kubectl patch deployment inventory -n default --type=json -p='[
    {"op":"remove","path":"/spec/template/spec/containers/0/resources"}
  ]' 2>/dev/null || true
  kubectl scale deployment/inventory -n default --replicas=1 >/dev/null 2>&1 || true
  kubectl rollout status deployment/inventory -n default --timeout=3m >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "==> Patch inventory resources (instrumentation.odigos.io/generic) — ns is already instrumented"
kubectl patch deployment inventory -n default --type=strategic -p='
spec:
  template:
    spec:
      containers:
      - name: inventory
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
            instrumentation.odigos.io/generic: "1"
          limits:
            instrumentation.odigos.io/generic: "1"
'

echo "==> Taint kind nodes CriticalAddonsOnly so scaled pods cannot schedule on kind"
taint_kind_nodes

echo "==> Scale inventory to 5"
kubectl scale deployment/inventory -n default --replicas=5

echo "==> Expect Pending inventory pods and no NodeClaim without NodeOverlay"
DEADLINE=$((SECONDS + 60))
PENDING_SEEN=0
PENDING=0
CLAIMS=0
while (( SECONDS < DEADLINE )); do
  PENDING=$(kubectl get pods -n default -l app=inventory --field-selector=status.phase=Pending -o name 2>/dev/null | wc -l | tr -d ' ')
  CLAIMS=$(kubectl get nodeclaims -o json 2>/dev/null | jq '.items | length')
  if [[ "${PENDING}" -ge 1 && "${CLAIMS}" -eq 0 ]]; then
    PENDING_SEEN=1
    break
  fi
  sleep 2
done

if [[ "${PENDING_SEEN}" -ne 1 ]]; then
  echo "❌ expected Pending inventory pod(s) and 0 NodeClaims before NodeOverlay"
  kubectl get pods -n default -l app=inventory -o wide || true
  kubectl describe pods -n default -l app=inventory | sed -n '/Events:/,$p' | head -40 || true
  kubectl get nodeclaims -o wide || true
  kubectl -n kube-system logs deploy/karpenter --tail=40 || true
  exit 1
fi
echo "✅ before NodeOverlay: Pending=${PENDING}, NodeClaims=${CLAIMS}"

sleep 15
CLAIMS=$(kubectl get nodeclaims -o json 2>/dev/null | jq '.items | length')
if [[ "${CLAIMS}" -ne 0 ]]; then
  echo "❌ Karpenter created NodeClaim(s) without NodeOverlay (count=${CLAIMS})"
  kubectl get nodeclaims -o wide
  exit 1
fi
echo "✅ still no NodeClaim without NodeOverlay"

echo "==> Apply NodeOverlay"
kubectl apply -f "${SCRIPT_DIR}/nodeoverlay.yaml"
kubectl wait --for=condition=Ready nodeoverlay/odigos-instrumentation --timeout=2m

echo "==> Expect Karpenter NodeClaim after NodeOverlay"
"${SCRIPT_DIR}/assert-nodeclaim-created.sh"

echo "==> Deleting NodeClaims/KWOK nodes before kindnet breaks"
delete_kwok_residue
repair_kindnet

echo "✅ NodeOverlay enabled Karpenter NodeClaim for instrumented inventory demand"
