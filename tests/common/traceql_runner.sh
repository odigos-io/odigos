#!/bin/bash

# Ensure the script fails if any command fails
set -e

# Function to perform the port-forward
function start_port_forward() {
  kubectl port-forward svc/e2e-tests-tempo 3100:3100 &
  PORT_FORWARD_PID=$!
}

# Function to stop the port-forward
function stop_port_forward() {
  kill $PORT_FORWARD_PID
}

# Function to verify the YAML schema
function verify_yaml_schema() {
  local file=$1
  local query=$(yq e '.query' "$file")
  local expected_count=$(yq e '.expected.count' "$file")
  echo "query is $query and expected_count is $expected_count"

  if [ -z "$query" ] || [ "$expected_count" == "null" ] || [ -z "$expected_count" ]; then
    echo "Invalid YAML schema in file: $file"
    exit 1
  fi
}

# Function to process a YAML file
function process_yaml_file() {
  local file=$1
  query=$(yq e -o=json '.query' "$file")
  expected_count=$(yq e '.expected.count' "$file")

  # Perform the HTTP request
  response=$(curl -s -X POST "http://localhost:3100/api/traces" -H "Content-Type: application/json" -d "$query")

  # Extract the actual count from the response (adjust this based on the actual structure of your response)
  actual_count=$(echo $response | jq '.data | length')

  # Compare the actual count with the expected count
  if [ "$actual_count" -ne "$expected_count" ]; then
    echo "Test failed for $file: expected $expected_count but got $actual_count"
    return 1
  else
    echo "Test passed for $file"
  fi
}

# Check if the first argument is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <directory>"
  exit 1
fi

# Directory containing the YAML files
DIRECTORY=$1

# Check if yq is installed
if ! command -v yq &> /dev/null; then
  echo "yq command not found. Please install yq."
  exit 1
fi

# Start port-forwarding
start_port_forward

# Trap to ensure the port-forward is stopped on script exit
trap stop_port_forward EXIT

# Process each YAML file in the directory
for file in "$DIRECTORY"/*.yaml; do
  echo "Processing $file"
  verify_yaml_schema $file
  process_yaml_file $file
done

echo "All tests passed."
