#!/bin/bash

# Ensure the script fails if any command fails
set -e

function flush_traces() {
  local dest_namespace="traces"
  local dest_service="e2e-tests-tempo"
  local dest_port="tempo-prom-metrics"
  kubectl get --raw /api/v1/namespaces/$dest_namespace/services/$dest_service:$dest_port/proxy/flush
  # check if command succeeded
  if [ $? -eq 0 ]; then
    echo "Traces flushed successfully"
  else
    echo "Failed to flush traces"
    exit 1
  fi
}

flush_traces