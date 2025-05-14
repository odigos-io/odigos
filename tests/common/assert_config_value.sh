#!/bin/bash

# Default values
CONFIG_MAP_NAME="effective-config"
NAMESPACE="odigos-test"

# Help message
usage() {
  echo "Usage: $0 -k <key> -e <expected_value> [-c <config_map_name>] [-n <namespace>]"
  echo
  echo "  -k <key>                 Dot-notated key path (e.g., .agentEnvVarsInjectionMethod)"
  echo "  -v <expected_value>      Expected value to assert"
  echo "  -c <config_map_name>     Optional, default: effective-config"
  echo "  -n <namespace>           Optional, default: odigos-test"
  exit 1
}

# Parse flags
while getopts ":k:v:c:n:" opt; do
  case $opt in
    k) KEY_PATH="$OPTARG" ;;
    v) EXPECTED_VALUE="$OPTARG" ;;
    c) CONFIG_MAP_NAME="$OPTARG" ;;
    n) NAMESPACE="$OPTARG" ;;
    *) usage ;;
  esac
done

# Ensure required arguments are present
if [[ -z "$KEY_PATH" || -z "$EXPECTED_VALUE" ]]; then
  usage
fi

# Generate full yq path by prepending dot internally
YQ_PATH=".$KEY_PATH"

# Extract and compare
ACTUAL_VALUE=$(kubectl get cm -n "$NAMESPACE" "$CONFIG_MAP_NAME" -o yaml 2>/dev/null | \
  yq ".data[\"config.yaml\"] | from_yaml | $YQ_PATH" 2>/dev/null)

if [[ "$ACTUAL_VALUE" == "$EXPECTED_VALUE" ]]; then
  echo "✅ Assertion passed: $KEY_PATH == $EXPECTED_VALUE"
else
  echo "❌ Assertion failed: Expected '$EXPECTED_VALUE' but got '$ACTUAL_VALUE'"
  exit 1
fi
