#!/bin/bash

# Ensure the script fails if any command fails
set -e

# Function to verify the YAML schema
function verify_yaml_schema() {
  local file=$1
  local query=$(yq e '.query' "$file")
  local expected_count=$(yq e '.expected.count' "$file")

  if [ -z "$query" ] || [ "$expected_count" == "null" ] || [ -z "$expected_count" ]; then
    echo "Invalid YAML schema in file: $file"
    exit 1
  fi
}

function urlencode() (
  local length="${#1}"
  for (( i = 0; i < length; i++ )); do
    local c="${1:i:1}"
    case $c in
      [a-zA-Z0-9.~_-]) printf "$c" ;;
      *) printf '%%%02X' "'$c" ;;
    esac
  done
)

# Function to process a YAML file
function process_yaml_file() {
  local dest_namespace="traces"
  local dest_service="e2e-tests-tempo"
  local dest_port="tempo-prom-metrics"

  local file=$1
  file_name=$(basename "$file")
  echo "Running test $file_name"
  query=$(yq '.query' "$file")
  encoded_query=$(urlencode "$query")
  expected_count=$(yq e '.expected.count' "$file")
  current_epoch=$(date +%s)
  one_hour=3600
  start_epoch=$(($current_epoch - one_hour))
  end_epoch=$(($current_epoch + one_hour))
  response=$(kubectl get --raw /api/v1/namespaces/$dest_namespace/services/$dest_service:$dest_port/proxy/api/search\?end=$end_epoch\&start=$start_epoch\&q=$encoded_query)
  num_of_traces=$(echo $response | jq '.traces | length')
  # if num_of_traces not equal to expected_count
  if [ "$num_of_traces" -ne "$expected_count" ]; then
    echo "Test FAILED: expected $expected_count got $num_of_traces"
    echo "$response" | jq
    return 1
  else
    echo "Test PASSED"
  fi
#  # Perform the HTTP request
#  response=$(curl -G -s "http://localhost:3100/api/search")
#  echo "$response"
#  # Extract the actual count from the response (adjust this based on the actual structure of your response)
#  actual_count=$(echo $response | jq '.data | length')
#
#  # Compare the actual count with the expected count
#  if [ "$actual_count" -ne "$expected_count" ]; then
#    echo "Test failed for $file: expected $expected_count but got $actual_count"
#    return 1
#  else
#    echo "Test passed for $file"
#  fi
}

# Check if the first argument is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <traceql-test-file>"
  exit 1
fi

# Test file path
TEST_FILE=$1

# Check if yq is installed
if ! command -v yq &> /dev/null; then
  echo "yq command not found. Please install yq."
  exit 1
fi

verify_yaml_schema $TEST_FILE
process_yaml_file $TEST_FILE
