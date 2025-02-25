#!/bin/bash

# Ensure the script fails if any command fails
set -e

# function to verify tempo is ready
# This is needed due to bug in tempo - It reports Ready before it is actually ready
# So we manually hit the health check endpoint to verify it is ready
wait_for_ready() {
  local namespace="traces"
  local pod_name="temp-curl-checker"
  local service_name="e2e-tests-tempo"  # service name as defined in your cluster
  local max_retries=30
  local retries=0

  echo "Creating temporary pod..."
  # Launch a pod that just sleeps for an hour (enough time for our check)
  kubectl run "$pod_name" -n "$namespace" --image=curlimages/curl --restart=Never --command -- sleep 3600 >/dev/null

  echo "Waiting for pod $pod_name to be ready..."
  kubectl wait --for=condition=Ready pod/"$pod_name" -n "$namespace" --timeout=60s

  while [ $retries -lt $max_retries ]; do
    echo "Checking readiness endpoint from within the pod..."
    # Execute curl inside the temporary pod to query the /ready endpoint via cluster DNS.
    response=$(kubectl exec -n "$namespace" "$pod_name" -- curl -s "http://${service_name}:3100/ready")
    if [ "$response" = "ready" ]; then
      echo "Tempo is ready."
      kubectl delete pod "$pod_name" -n "$namespace" --ignore-not-found >/dev/null
      return 0
    else
      echo "Tempo is not ready yet. Retrying in 2 seconds..."
      sleep 2
      retries=$((retries+1))
    fi
  done

  echo "Tempo did not become ready within the expected time."
  kubectl delete pod "$pod_name" -n "$namespace" --ignore-not-found >/dev/null
  return 1
}

wait_for_ready