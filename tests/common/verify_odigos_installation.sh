#!/bin/bash

# Default namespace
NAMESPACE=${1:-odigos-test}
TIMEOUT="4m"

echo "Verifying Odigos installation in namespace: $NAMESPACE"

# Function to check if a command succeeded
check_command() {
    if [ $? -eq 0 ]; then
        echo "✅ $1"
    else
        echo "❌ $1"
        exit 1
    fi
}

kubectl wait --for=create deployment/odigos-instrumentor -n $NAMESPACE --timeout=$TIMEOUT
check_command "Odigos Instrumentor deployment created"

kubectl wait --for=create deployment/odigos-autoscaler -n $NAMESPACE --timeout=$TIMEOUT
check_command "Odigos Autoscaler deployment created"

kubectl wait --for=create deployment/odigos-scheduler -n $NAMESPACE --timeout=$TIMEOUT
check_command "Odigos Scheduler deployment created"

kubectl wait --for=create deployment/odigos-ui -n $NAMESPACE --timeout=$TIMEOUT
check_command "Odigos UI deployment created"

kubectl wait --for=create daemonset/odiglet -n $NAMESPACE --timeout=$TIMEOUT
check_command "Odiglet DaemonSet created"

# Wait for pods to be created first
until kubectl get pods -n $NAMESPACE 2>/dev/null | grep -q .; do
    sleep 2
done

# Now wait for pods to be ready
kubectl wait --for=condition=ready pods --all -n $NAMESPACE --timeout=$TIMEOUT
check_command "All pods are ready"

echo "✅ Odigos installation verification completed successfully" 