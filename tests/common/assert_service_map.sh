#!/bin/bash

set -e

EXPECTED_FILE=$1
LOCAL_PORT=$(shuf -i 20000-65000 -n 1)

kubectl port-forward svc/ui "$LOCAL_PORT":3000 -n odigos-test &
PF_PID=$!
trap "kill $PF_PID 2>/dev/null || true" EXIT
sleep 3

EXPECTED=$(jq -S '.' "$EXPECTED_FILE")

while true; do
    ACTUAL=$(curl -s -X POST "http://localhost:$LOCAL_PORT/graphql" \
        -H "Content-Type: application/json" \
        -d '{
          "operationName": "GetServiceMap",
          "variables": {},
          "query": "query GetServiceMap { getServiceMap { services { nodeId serviceName services { nodeId isVirtual serviceName requests dateTime nodeAttributes { key value } } } } }"
        }' | jq -S '
          .data.getServiceMap.services
          | map(del(.nodeId) | .services |= (map(del(.nodeId, .dateTime, .requests)) | sort_by(.serviceName)))
          | sort_by(.serviceName)
        ' 2>/dev/null) || true

    if [[ -n "$ACTUAL" && "$ACTUAL" == "$EXPECTED" ]]; then
        echo "Service map matches expected!"
        exit 0
    fi

    echo "Service map not yet matching, retrying..."
    echo "Actual: $ACTUAL"
    sleep 3
done
