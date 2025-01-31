#!/bin/bash

# Check if jq is installed
if ! command -v jq &> /dev/null; then
  echo "qq command not found. Please install qq."
  exit 1
fi

payload='{
  "operationName": "GetOverviewMetrics",
  "variables": {},
  "query": "query GetOverviewMetrics { getOverviewMetrics { sources { namespace kind name totalDataSent throughput } destinations { id totalDataSent throughput } } }"
}'

# Send the GraphQL request and store the response
response=$(curl -s -X POST http://localhost:3000/graphql \
    -H "Content-Type: application/json" \
    -d "$payload")

# Check if the response contains an error message
error_message=$(echo "$response" | jq -r '.errors[0].message // empty')
if [[ -n "$error_message" ]]; then
    echo "❌ GraphQL Error: $error_message"
    exit 1
fi

# Ensure the response contains valid data
if [[ $(echo "$response" | jq '.data.getOverviewMetrics') == "null" ]]; then
    echo "❌ Error: Missing 'getOverviewMetrics' data in the response."
    exit 1
fi

# Validate sources totalDataSent
invalid_sources=$(echo "$response" | jq '.data.getOverviewMetrics.sources[] | select(.totalDataSent <= 0)')

# Validate destinations totalDataSent
invalid_destinations=$(echo "$response" | jq '.data.getOverviewMetrics.destinations[] | select(.totalDataSent <= 0)')

# Check if any source has invalid totalDataSent
if [[ -n "$invalid_sources" ]]; then
    echo "⚠️ ERROR: Some sources have non-positive totalDataSent:"
    echo "$invalid_sources" | jq .
    exit 1
fi

# Check if any destination has invalid totalDataSent
if [[ -n "$invalid_destinations" ]]; then
    echo "⚠️ ERROR: Some destinations have non-positive totalDataSent:"
    echo "$invalid_destinations" | jq .
    exit 1
fi

# If everything is valid
echo "✅ All sources and destinations have valid totalDataSent values."
exit 0
