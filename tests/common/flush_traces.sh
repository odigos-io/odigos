#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/curl_helper.sh"

# Ensure the script fails if any command fails
set -e

function flush_traces() {
  local dest_namespace="traces"
  local dest_service="e2e-tests-tempo"
  local dest_port=3100
  
  deploy_curl_pod $dest_namespace
  run_curl_cmd $dest_namespace http://${dest_service}:${dest_port}/flush
  delete_curl_pod $dest_namespace

  # check if command succeeded
  if [ $? -eq 0 ]; then
    echo "Traces flushed successfully"
  else
    echo "Failed to flush traces"
    exit 1
  fi
}

flush_traces