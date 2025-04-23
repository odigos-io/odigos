#!/bin/bash

kind create cluster || {
  echo "Error: Failed to create Kind cluster"
  exit 1
}

# Build CLI
go build -tags=embed_manifests -o odigos-e2e-test cli/main.go

# Build and Load Odigos Images
TAG=e2e-test make build-images load-to-kind

# Install Odigos
./odigos-e2e-test install --version e2e-test

# Install Collector - Add Dependencies
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts

if [ ! -d "opentelemetry-helm-charts" ]; then
  git clone https://github.com/open-telemetry/opentelemetry-helm-charts.git
fi

# Install Collector
helm install test -f .github/workflows/e2e/collector-helm-values.yaml opentelemetry-helm-charts/charts/opentelemetry-collector --namespace traces --create-namespace

# Wait for Collector to be ready
echo "Waiting for Collector to be ready..."
kubectl wait --for=condition=Ready --timeout=60s -n traces pod/test-opentelemetry-collector-0

# Install KV Shop
kubectl create ns kvshop
kubectl apply -f .github/workflows/e2e/kv-shop.yaml -n kvshop

# Wait for KV Shop to be ready
echo "Waiting for KV Shop to be ready..."
kubectl wait --for=condition=Ready --timeout=100s -n kvshop pods --all

# Select kvshop namespace for instrumentation
kubectl label namespace kvshop odigos-instrumentation=enabled

# Connect to Jaeger destination
kubectl create -f .github/workflows/e2e/jaeger-dest.yaml

# Wait for Odigos to bring up collectors
while [[ $(kubectl get daemonset odigos-data-collection -n odigos-system -o jsonpath='{.status.numberReady}') != 1 ]]; do
  echo "Waiting for odigos-data-collection daemonset to be created" && sleep 3
done
while [[ $(kubectl get deployment odigos-gateway -n odigos-system -o jsonpath='{.status.readyReplicas}') != 1 ]]; do
  echo "Waiting for odigos-data-collection deployment to be created" && sleep 3
done
while [[ $(kubectl get pods -n kvshop | grep Running | wc -l) -ne 5 ]]; do
  echo "Waiting for kvshop pods to be running" && sleep 3
done
sleep 10
kubectl get pods -A
kubectl get svc -A

# Start bot job
kubectl create -f .github/workflows/e2e/traffic-bot.yaml -n kvshop

# Wait for bot job to complete
echo "Waiting for bot job to complete..."
kubectl wait --for=condition=Complete --timeout=60s job/traffic-bot -n kvshop

# Copy trace output
echo "Sleeping for 10 seconds to allow traces to be collected"
sleep 10
kubectl cp -c filecp traces/test-opentelemetry-collector-0:tmp/trace.json ./.github/workflows/e2e/bats/traces-orig.json

# Verify output trace
bats .github/workflows/e2e/bats/verify.bats
