#!/bin/bash

DEFAULT_NAMESPACE="odigos-test"
DEFAULT_TIMEOUT="120s"
CHECK_INTERVAL=5 

NAMESPACE=${1:-$DEFAULT_NAMESPACE}
TIMEOUT=${2:-$DEFAULT_TIMEOUT}

# Define expected labels for pods (adjust as needed)
EXPECTED_LABELS=(
  "odigos.io/collector-role=NODE_COLLECTOR" # For odigos-data-collection pods
  "odigos.io/collector-role=CLUSTER_GATEWAY" # For odigos-gateway pods
  "app.kubernetes.io/name=odigos-autoscaler"
  "app.kubernetes.io/name=odigos-instrumentor"
  "app.kubernetes.io/name=odigos-scheduler"
  "app=odigos-ui"
)

echo "Waiting for all expected pods to be ready in namespace '$NAMESPACE' with timeout of $TIMEOUT..."

for label in "${EXPECTED_LABELS[@]}"; do
  echo "Waiting for pods with label: $label..."
  # Wait for pods to exist before using kubectl wait
  EXISTS=false
  while [[ "$EXISTS" == "false" ]]; do
    POD_COUNT=$(kubectl get pods -l "$label" -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
    if [[ "$POD_COUNT" -gt 0 ]]; then
      EXISTS=true
      echo "Found $POD_COUNT pod(s) with label '$label'. Proceeding to wait for readiness..."
    else
      echo "No pods found with label '$label'. Checking again in $CHECK_INTERVAL seconds..."
      sleep $CHECK_INTERVAL
    fi
  done

  # Use `kubectl wait` to check all pods matching the label
  kubectl wait --for=condition=Ready pods -l "$label" -n "$NAMESPACE" --timeout="$TIMEOUT"
  if [[ $? -ne 0 ]]; then
    echo "Pods with label '$label' did not become ready within $TIMEOUT in namespace '$NAMESPACE'"
    exit 1
  fi
done

echo "All expected pods are ready!"
