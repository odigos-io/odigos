#!/bin/bash

# Default namespace
NAMESPACE=${1:-odigos-test}
TIMEOUT="4m"

echo "Verifying Odigos installation in namespace: $NAMESPACE"

FAIL=0

# Wait for all resource creation in parallel
wait_for_create() {
    local resource=$1
    local name=$2
    if kubectl wait --for=create "$resource/$name" -n "$NAMESPACE" --timeout="$TIMEOUT"; then
        echo "✅ $name created"
    else
        echo "❌ $name not created"
        FAIL=1
    fi
}

wait_for_create deployment odigos-instrumentor &
wait_for_create deployment odigos-autoscaler &
wait_for_create deployment odigos-scheduler &
wait_for_create deployment odigos-ui &
wait_for_create daemonset odiglet &
wait

if [ $FAIL -ne 0 ]; then
    echo "❌ One or more resources failed to be created"
    exit 1
fi

# Wait for pods to be created first
until kubectl get pods -n $NAMESPACE 2>/dev/null | grep -q .; do
    sleep 2
done

# Now wait for pods to be ready
if kubectl wait --for=condition=ready pods --all -n $NAMESPACE --timeout=$TIMEOUT; then
    echo "✅ All pods are ready"
else
    echo "❌ Not all pods are ready"
    exit 1
fi

echo "✅ Odigos installation verification completed successfully" 