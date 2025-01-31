#!/bin/bash

# Check if jq is installed
if ! command -v jq &> /dev/null; then
  echo "qq command not found. Please install qq."
  exit 1
fi

print_help() {
    echo "Usage: $0 [namespace] <valid_sources> <valid_destinations>"
    echo ""
    echo "Arguments:"
    echo "  namespace       (optional) The Kubernetes namespace of the UI service (default: 'odigos-system')"
    echo "  valid_sources   (required) The minimum number of valid sources expected"
    echo "  valid_destinations (required) The minimum number of valid destinations expected"
    echo ""
    echo "Example:"
    echo "  $0 odigos-system 5 2   # Uses 'odigos-system' as the namespace, expects 5 valid sources & 2 valid destinations"
    echo "  $0 5 2                 # Uses default namespace 'odigos-system', expects 5 valid sources & 2 valid destinations"
    exit 1
}

if [[ $# -lt 2 || $# -gt 3 ]]; then
    echo "❌ Error: Invalid number of arguments."
    print_help
fi

if [[ $# -eq 3 ]]; then
    NAMESPACE=$1
    VALID_SOURCES=$2
    VALID_DESTINATIONS=$3
else
    NAMESPACE="odigos-system"
    VALID_SOURCES=$1
    VALID_DESTINATIONS=$2
fi

echo "ℹ️ Using namespace: $NAMESPACE"
echo "ℹ️ Expecting at least $VALID_SOURCES valid sources and $VALID_DESTINATIONS valid destinations"

find_free_port() {
    while :; do
        local port=$(shuf -i 20000-65000 -n 1)
        if ! nc -z localhost $port 2>/dev/null; then
            echo "$port"
            return
        fi
    done
}

cleanup() {
    if [[ -n "$PORT_FORWARD_PID" ]]; then
        echo "Cleaning up: Stopping port-forward (PID: $PORT_FORWARD_PID)"
        kill $PORT_FORWARD_PID 2>/dev/null
    fi
}
trap cleanup EXIT

# Try up to 5 different ports before giving up
for attempt in {1..5}; do
    LOCAL_PORT=$(find_free_port)
    echo "🔀 Attempt $attempt: Trying port $LOCAL_PORT..."

    kubectl port-forward svc/ui $LOCAL_PORT:3000 -n "$NAMESPACE" & PORT_FORWARD_PID=$!

    retry_delay=0.1
    for i in {1..10}; do
        if nc -z localhost $LOCAL_PORT 2>/dev/null; then
            echo "✅ Successfully established port-forward on port $LOCAL_PORT"
            break 2 # Break out of both loops
        fi
        sleep $retry_delay
    done

    echo "❌ Failed to establish port-forward on port $LOCAL_PORT. Retrying..."
    cleanup
done

# If no successful port-forward, exit with error
if ! nc -z localhost $LOCAL_PORT 2>/dev/null; then
    echo "❌ Error: Unable to establish port-forward after multiple attempts."
    exit 1
fi

grahphqlPayload='{
  "operationName": "GetOverviewMetrics",
  "variables": {},
  "query": "query GetOverviewMetrics { getOverviewMetrics { sources { namespace kind name totalDataSent throughput } destinations { id totalDataSent throughput } } }"
}'

# Send the GraphQL request and store the response
response=$(curl -s -X POST http://localhost:$LOCAL_PORT/graphql \
    -H "Content-Type: application/json" \
    -d "$grahphqlPayload")

if [[ -z "$response" ]]; then
    echo "❌ Error: Empty response from server."
    exit 1
fi

error_message=$(echo "$response" | jq -r '.errors[0].message // empty')
if [[ -n "$error_message" ]]; then
    echo "❌ GraphQL Error: $error_message"
    exit 1
fi

if [[ $(echo "$response" | jq '.data.getOverviewMetrics') == "null" ]]; then
    echo "❌ Error: Missing 'getOverviewMetrics' data in the response."
    exit 1
fi

valid_sources_count=$(echo "$response" | jq '.data.getOverviewMetrics.sources | map(select(.totalDataSent > 0)) | length')
valid_destinations_count=$(echo "$response" | jq '.data.getOverviewMetrics.destinations | map(select(.totalDataSent > 0)) | length')

if [[ "$valid_sources_count" -lt "$VALID_SOURCES" ]]; then
    echo "❌ Error: Expected at least $VALID_SOURCES valid sources, but found only $valid_sources_count."
    exit 1
fi

if [[ "$valid_destinations_count" -lt "$VALID_DESTINATIONS" ]]; then
    echo "❌ Error: Expected at least $VALID_DESTINATIONS valid destinations, but found only $valid_destinations_count."
    exit 1
fi

echo "✅ All checks passed: Sources ($valid_sources_count) and Destinations ($valid_destinations_count) meet the expected criteria."
exit 0
