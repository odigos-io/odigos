#!/bin/bash

# Arguments with defaults
NAMESPACE=${1:-odigos-test}
TIMEOUT="10m"

echo "Waiting for NodeDetails to be created in namespace: $NAMESPACE with timeout: $TIMEOUT"

# Convert timeout to seconds (e.g., "4m" -> 240)
TIMEOUT_SECONDS=$(echo $TIMEOUT | sed 's/m$//' | awk '{print $1 * 60}')
if [ -z "$TIMEOUT_SECONDS" ] || [ "$TIMEOUT_SECONDS" -eq 0 ]; then
    TIMEOUT_SECONDS=240
fi

elapsed=0
echo "Waiting for at least one NodeDetails resource to be created..."

while [ $elapsed -lt $TIMEOUT_SECONDS ]; do
    # Check if at least one NodeDetails exists
    count=$(kubectl get nodedetailses -n $NAMESPACE --no-headers 2>/dev/null | wc -l)

    if [ "$count" -gt 0 ]; then
        echo "✅ Found $count NodeDetails resource(s)"

        # Print details
        kubectl get nodedetailses -n $NAMESPACE

        exit 0
    fi

    sleep 2
    elapsed=$((elapsed + 2))

    # Show progress every 10 seconds
    if [ $((elapsed % 10)) -eq 0 ]; then
        echo "Still waiting for NodeDetails... (${elapsed}s elapsed)"
    fi
done

echo "❌ Timeout waiting for NodeDetails after ${TIMEOUT_SECONDS}s"
echo "Current NodeDetails in namespace $NAMESPACE:"
kubectl get nodedetailses -n $NAMESPACE 2>&1

exit 1

