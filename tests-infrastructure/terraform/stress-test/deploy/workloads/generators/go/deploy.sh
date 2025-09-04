#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Deploying Go span generator to Kubernetes...${NC}"

# Check if namespace exists, create if not
kubectl get namespace load-test >/dev/null 2>&1 || kubectl create namespace load-test

# Deploy the deployment
kubectl apply -f deployment.yaml

echo -e "${GREEN}Deployment completed successfully!${NC}"

# Wait for deployment to be ready
echo -e "${YELLOW}Waiting for deployment to be ready...${NC}"
kubectl wait --for=condition=available --timeout=300s deployment/go-span-generator -n load-test

echo -e "${GREEN}Go span generator is now running!${NC}"

# Show deployment status
echo -e "${YELLOW}Deployment status:${NC}"
kubectl get pods -n load-test -l app=go-span-generator

echo -e "${YELLOW}Service status:${NC}"
kubectl get svc -n load-test -l app=go-span-generator
