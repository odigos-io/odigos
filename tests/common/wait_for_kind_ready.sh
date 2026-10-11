#!/usr/bin/env bash
# Wait until a Kind cluster is actually usable for E2E APPLY/install steps.
#
# Kind/helm-kind-action can report Ready while kube-apiserver/etcd are still
# flapping under runner load (especially k8s 1.32). Node/pod Ready + CoreDNS
# rollout is necessary but not sufficient — early kubectl/chainsaw writes then
# fail with "context deadline exceeded" / "etcdserver: request timed out"
# (DEVOPS-84).
set -euo pipefail

READYZ_REQUIRED_SUCCESSES="${READYZ_REQUIRED_SUCCESSES:-5}"
READYZ_TIMEOUT_SECONDS="${READYZ_TIMEOUT_SECONDS:-180}"
READYZ_SLEEP_SECONDS="${READYZ_SLEEP_SECONDS:-2}"
WRITE_PROBE_TIMEOUT_SECONDS="${WRITE_PROBE_TIMEOUT_SECONDS:-60}"
WRITE_PROBE_NAMESPACE="${WRITE_PROBE_NAMESPACE:-odigos-kind-ready-probe}"

echo "Waiting for nodes, kube-system pods, and CoreDNS..."
kubectl wait --for=condition=Ready node --all --timeout=180s
kubectl wait -n kube-system --for=condition=Ready pod --all --timeout=300s
kubectl rollout status -n kube-system deployment/coredns --timeout=300s

echo "Waiting for kube-apiserver /readyz (${READYZ_REQUIRED_SUCCESSES} consecutive successes, timeout ${READYZ_TIMEOUT_SECONDS}s)..."
successes=0
deadline=$((SECONDS + READYZ_TIMEOUT_SECONDS))
while (( successes < READYZ_REQUIRED_SUCCESSES )); do
  if (( SECONDS >= deadline )); then
    echo "ERROR: kube-apiserver /readyz did not stay healthy for ${READYZ_REQUIRED_SUCCESSES} consecutive checks within ${READYZ_TIMEOUT_SECONDS}s"
    kubectl get --raw='/readyz?verbose' || true
    kubectl get --raw='/livez?verbose' || true
    kubectl get nodes -o wide || true
    kubectl get pods -n kube-system -o wide || true
    exit 1
  fi

  if kubectl get --raw='/readyz?verbose' >/tmp/odigos-kind-readyz.out 2>/tmp/odigos-kind-readyz.err; then
    successes=$((successes + 1))
    echo "readyz ok (${successes}/${READYZ_REQUIRED_SUCCESSES})"
  else
    successes=0
    echo "readyz not ready; resetting consecutive success counter"
    cat /tmp/odigos-kind-readyz.err >&2 || true
  fi
  sleep "${READYZ_SLEEP_SECONDS}"
done

echo "Probing API write path (create/delete namespace)..."
write_deadline=$((SECONDS + WRITE_PROBE_TIMEOUT_SECONDS))
until kubectl create namespace "${WRITE_PROBE_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>/tmp/odigos-kind-write-probe.err; do
  if (( SECONDS >= write_deadline )); then
    echo "ERROR: API write probe failed within ${WRITE_PROBE_TIMEOUT_SECONDS}s"
    cat /tmp/odigos-kind-write-probe.err >&2 || true
    exit 1
  fi
  echo "write probe not ready; retrying..."
  cat /tmp/odigos-kind-write-probe.err >&2 || true
  sleep "${READYZ_SLEEP_SECONDS}"
done
kubectl delete namespace "${WRITE_PROBE_NAMESPACE}" --wait=false >/dev/null 2>&1 || true

echo "Kind control plane is ready for E2E"
