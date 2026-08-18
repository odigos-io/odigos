#!/usr/bin/env bash
set -euo pipefail

P=${1:-"../../.."}
ROOT="$(cd "$P" && pwd)"
CRD="odigoscloudconnectors.odigos.io"
RUN_ID="${RANDOM}${RANDOM}"
RELEASE="crd-migration-$RUN_ID"
NAMESPACE="crd-migration-$RUN_ID"
TMP_DIR="$(mktemp -d)"

cleanup() {
  kubectl delete odigoscloudconnector migration-canary --namespace "$NAMESPACE" --ignore-not-found --wait=true >/dev/null 2>&1 || true
  helm uninstall "$RELEASE" --namespace "$NAMESPACE" >/dev/null 2>&1 || true
  kubectl delete namespace "$NAMESPACE" --ignore-not-found --wait=false >/dev/null 2>&1 || true
  kubectl delete crd "$CRD" --ignore-not-found --wait=true >/dev/null 2>&1 || true
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

if kubectl get "crd/$CRD" >/dev/null 2>&1; then
  echo "Refusing to replace existing CRD $CRD"
  exit 1
fi

for stage in legacy protected removed; do
  mkdir -p "$TMP_DIR/$stage/templates"
  cp "$ROOT/helm/odigos/Chart.yaml" "$TMP_DIR/$stage/Chart.yaml"
done

sed '/helm.sh\/resource-policy: keep/d' \
  "$ROOT/helm/odigos/templates/crds/odigos.io_odigoscloudconnectors.yaml" \
  > "$TMP_DIR/legacy/templates/odigoscloudconnectors.yaml"
cp "$ROOT/helm/odigos/templates/crds/odigos.io_odigoscloudconnectors.yaml" \
  "$TMP_DIR/protected/templates/odigoscloudconnectors.yaml"

helm install "$RELEASE" "$TMP_DIR/legacy" --namespace "$NAMESPACE" --create-namespace >/dev/null
kubectl wait --for=condition=Established "crd/$CRD" --timeout=60s >/dev/null
kubectl apply --namespace "$NAMESPACE" -f - >/dev/null <<EOF
apiVersion: odigos.io/v1alpha1
kind: OdigosCloudConnector
metadata:
  name: migration-canary
spec:
  provider: aws
  account:
    id: migration-test
  credentialsSecretRef:
    name: unused
EOF

crd_uid="$(kubectl get "crd/$CRD" -o jsonpath='{.metadata.uid}')"
cr_uid="$(kubectl get odigoscloudconnector migration-canary --namespace "$NAMESPACE" -o jsonpath='{.metadata.uid}')"

helm upgrade "$RELEASE" "$TMP_DIR/protected" --namespace "$NAMESPACE" >/dev/null

protected_crd_uid="$(kubectl get "crd/$CRD" -o jsonpath='{.metadata.uid}')"
protected_cr_uid="$(kubectl get odigoscloudconnector migration-canary --namespace "$NAMESPACE" -o jsonpath='{.metadata.uid}')"
protected_crd_deletion="$(kubectl get "crd/$CRD" -o jsonpath='{.metadata.deletionTimestamp}')"
protected_cr_deletion="$(kubectl get odigoscloudconnector migration-canary --namespace "$NAMESPACE" -o jsonpath='{.metadata.deletionTimestamp}')"
resource_policy="$(kubectl get "crd/$CRD" -o jsonpath='{.metadata.annotations.helm\.sh/resource-policy}')"

test "$protected_crd_uid" = "$crd_uid"
test "$protected_cr_uid" = "$cr_uid"
test -z "$protected_crd_deletion"
test -z "$protected_cr_deletion"
test "$resource_policy" = "keep"

helm upgrade "$RELEASE" "$TMP_DIR/removed" --namespace "$NAMESPACE" >/dev/null

removed_crd_uid="$(kubectl get "crd/$CRD" -o jsonpath='{.metadata.uid}')"
removed_cr_uid="$(kubectl get odigoscloudconnector migration-canary --namespace "$NAMESPACE" -o jsonpath='{.metadata.uid}')"
removed_crd_deletion="$(kubectl get "crd/$CRD" -o jsonpath='{.metadata.deletionTimestamp}')"
removed_cr_deletion="$(kubectl get odigoscloudconnector migration-canary --namespace "$NAMESPACE" -o jsonpath='{.metadata.deletionTimestamp}')"
release_status="$(helm status "$RELEASE" --namespace "$NAMESPACE" | awk '/^STATUS:/ {print $2}')"

test "$removed_crd_uid" = "$crd_uid"
test "$removed_cr_uid" = "$cr_uid"
test -z "$removed_crd_deletion"
test -z "$removed_cr_deletion"
test "$release_status" = "deployed"

echo "Cloud connector CRD and custom resources survive both migration stages."
