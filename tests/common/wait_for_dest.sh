#!/bin/bash

# Ensure the script fails if any command fails
set -e

# function to verify tempo is ready
# This is needed due to bug in tempo - It reports Ready before it is actually ready
# So we manually hit the health check endpoint to verify it is ready
wait_for_ready() {
  local dest_namespace="traces"
  local dest_service="e2e-tests-tempo"
  local dest_port=3100  # numeric port from the service spec
  local max_retries=30
  local retries=3

  while [ $retries -lt $max_retries ]; do
    # Start port-forwarding the service locally in the background.
    kubectl port-forward svc/e2e-tests-tempo -n traces 3100:3100 >/dev/null 2>&1 &
    PF_PID=$!
    # Give the port-forward a moment to start.
    sleep 1

    # Query the /ready endpoint locally.
    local response
    response=$(curl -s "http://localhost:3100/ready")
    # Kill the port-forward process.
    kill $PF_PID >/dev/null 2>&1

    if [ "$response" = "ready" ]; then
      echo "Tempo is ready."
      return 0
    else
      echo "Tempo is not ready yet. Retrying in 2 seconds..."
      sleep 2
      retries=$((retries+1))
    fi
  done

  echo "Tempo did not become ready within the expected time."
  return 1
}

wait_for_ready