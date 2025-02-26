#!/bin/bash

# Ensure the script fails if any command fails
set -e

source "$(dirname "${BASH_SOURCE[0]}")/curl_helper.sh"

# function to verify tempo is ready
# This is needed due to bug in tempo - It reports Ready before it is actually ready
# So we manually hit the health check endpoint to verify it is ready
dest_namespace="traces"

function wait_for_ready() {
  local dest_service="e2e-tests-tempo"
  local dest_port=3100

  
  response=$(run_curl_cmd $dest_namespace "http://${dest_service}:${dest_port}/ready")
  

  if [ "$response" != "ready" ]; then
    echo "Tempo is not ready yet. Retrying in 2 seconds..."
    sleep 2
    wait_for_ready
  else
    echo "Tempo is ready"
    sleep 2
  fi
}

deploy_curl_pod $dest_namespace
wait_for_ready
delete_curl_pod $dest_namespace