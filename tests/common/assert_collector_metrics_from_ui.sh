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

payload='{
  "operationName": "GetOverviewMetrics",
  "variables": {},
  "query": "query GetOverviewMetrics { getOverviewMetrics { sources { namespace kind name totalDataSent throughput } destinations { id totalDataSent throughput } } }"
}'

kubectl port-forward svc/ui 3000:3000 -n "$NAMESPACE" &
PORT_FORWARD_PID=$!

cleanup() {
    echo "Cleaning up: Stopping port-forward (PID: $PORT_FORWARD_PID)"
    kill $PORT_FORWARD_PID
}

# Register cleanup function to run on script exit
trap cleanup EXIT

echo "⏳ Waiting for port 3000 to be available..."
for i in {1..10}; do
    if nc -z localhost 3000; then
        echo "✅ Port 3000 is ready!"
        break
    fi
    echo "🔄 Port 3000 not ready yet, retrying in 100 milliseconds..."
    sleep 0.1
done

# Send the GraphQL request and store the response
response=$(curl -s -X POST http://localhost:3000/graphql \
    -H "Content-Type: application/json" \
    -d "$payload")

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
