#!/bin/bash

# Ensure the script fails if any command fails
set -e

# function to verify tempo is ready
function wait_for_ready() {
  local dest_namespace="traces"
  local dest_service="e2e-tests-tempo"
  local dest_port="tempo-prom-metrics"
  local response=$(kubectl get --raw /api/v1/namespaces/$dest_namespace/services/$dest_service:$dest_port/proxy/ready)
  if [ "$response" != "ready" ]; then
    echo "Tempo is not ready yet. Retrying in 2 seconds..."
    sleep 2
    wait_for_ready
  fi
}

wait_for_ready