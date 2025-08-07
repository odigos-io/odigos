#!/bin/bash

# Check if jq is installed
if ! command -v jq &> /dev/null; then
  echo "jq command not found. Please install jq."
  exit 1
fi

print_help() {
    echo "Usage: $0 [namespace]"
    echo ""
    echo "Arguments:"
    echo "  namespace       (optional) The Kubernetes namespace of the UI service (default: 'odigos-system')"
    exit 1
}

if [[ $# -gt 1 ]]; then
    echo "❌ Error: Invalid number of arguments."
    print_help
fi

if [[ $# -eq 1 ]]; then
    NAMESPACE=$1
else
    NAMESPACE="odigos-system"
fi

echo "ℹ️ Using namespace: $NAMESPACE"

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

    kubectl port-forward svc/ui $LOCAL_PORT:3000 -n "$NAMESPACE" &
    PORT_FORWARD_PID=$!

    # Allow `kubectl port-forward` to finish establishing connection before tring to connect to it.
    sleep 5

    retry_delay=0.5
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



grahphqlOverviewPayload='{
  "operationName": "GetOverviewMetrics",
  "variables": {},
  "query": "query GetOverviewMetrics { getOverviewMetrics { sources { namespace kind name totalDataSent throughput } destinations { id totalDataSent throughput } } }"
}'

# Send the GraphQL request and store the response
response=$(curl -s -X POST http://localhost:$LOCAL_PORT/graphql \
    -H "Content-Type: application/json" \
    -d "$grahphqlOverviewPayload")

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

# check that we have at least one valid source and destination
# trying to count the number of sources appeared in the response and asserting that number can lead to flaky tests
valid_sources_count=$(echo "$response" | jq '.data.getOverviewMetrics.sources | map(select(.totalDataSent > 0)) | length')
valid_destinations_count=$(echo "$response" | jq '.data.getOverviewMetrics.destinations | map(select(.totalDataSent > 0)) | length')

if [[ "$valid_sources_count" -lt "1" ]]; then
    echo "❌ Error: Expected at least 1 valid source metrics, but found only $valid_sources_count."
    exit 1
fi

if [[ "$valid_destinations_count" -lt "1" ]]; then
    echo "❌ Error: Expected at least 1 valid destinations metrics, but found only $valid_destinations_count."
    exit 1
fi

grahphqlServiceGraphPayload='{
  "operationName": "GetServiceMap",
  "variables": {},
  "query": "query GetServiceMap { getServiceMap { services { serviceName services { serviceName requests dateTime } } } }"
}'

# Send the GraphQL request and store the response
response=$(curl -s -X POST http://localhost:$LOCAL_PORT/graphql \
    -H "Content-Type: application/json" \
    -d "$grahphqlServiceGraphPayload")

echo "🔍 Service Graph Response: $response"

if [[ -z "$response" ]]; then
    echo "❌ Error: Empty response from server."
    exit 1
fi

# Check for GraphQL errors
error_message=$(echo "$response" | jq -r '.errors[0].message // empty')
if [[ -n "$error_message" ]]; then
    echo "❌ GraphQL Error: $error_message"
    exit 1
fi

# Validate presence of getServiceMap.services
services_node=$(echo "$response" | jq '.data.getServiceMap.services')
if [[ "$services_node" == "null" || -z "$services_node" ]]; then
    echo "❌ Error: Missing 'getServiceMap.services' in the response."
    exit 1
fi


# Assert total number of top-level services [user, frontend, coupon]
service_count=$(echo "$services_node" | jq 'length')
if [[ "$service_count" -ne 3 ]]; then
    echo "❌ Error: Expected 3 top-level services, found $service_count."
    exit 1
fi

# Try to get the downstream count for 'frontend'
frontend_count=$(echo "$services_node" | jq '[.[] | select(.serviceName == "frontend") | .services | length] | first')

# If not found, fallback to 'frontend-reported', e.g for the source e2e test
if [[ -z "$frontend_count" || "$frontend_count" == "null" ]]; then
    frontend_count=$(echo "$services_node" | jq '[.[] | select(.serviceName == "frontend-reported") | .services | length] | first')
fi

if [[ -z "$frontend_count" || "$frontend_count" == "null" ]]; then
    echo "❌ Error: Neither 'frontend' nor 'frontend-reported' services found in Service Graph."
    exit 1
fi

if [[ "$frontend_count" -ne 5 ]]; then
    echo "❌ Error: Expected 'frontend' or 'frontend-reported' to have 5 downstream services, found $frontend_count."
    exit 1
fi


echo "✅ All checks passed: Sources ($valid_sources_count), Destinations ($valid_destinations_count), and Service Graph structure (frontend with $frontend_count downstream services) meet the expected criteria."
exit 0
