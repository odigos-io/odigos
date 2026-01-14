#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ECR repository details
ECR_REPO="public.ecr.aws/odigos"
IMAGE_NAME="python-span-gen"
TAG="v0.0.1"

FULL_IMAGE_NAME="${ECR_REPO}/${IMAGE_NAME}:${TAG}"

echo -e "${YELLOW}Building and pushing Python span generator to ECR...${NC}"

# Build and push multi-platform image to ECR
docker buildx build --push \
  --platform linux/amd64,linux/arm64 \
  -t "${FULL_IMAGE_NAME}" \
  -f Dockerfile .

echo -e "${GREEN}Successfully pushed ${FULL_IMAGE_NAME} to ECR!${NC}"

# Also tag as latest
LATEST_IMAGE_NAME="${ECR_REPO}/${IMAGE_NAME}:latest"
echo -e "${YELLOW}Tagging as latest...${NC}"

docker buildx build --push \
  --platform linux/amd64,linux/arm64 \
  -t "${LATEST_IMAGE_NAME}" \
  -f Dockerfile .

echo -e "${GREEN}Successfully pushed ${LATEST_IMAGE_NAME} to ECR!${NC}"

echo -e "${GREEN}Build and push completed!${NC}"
echo -e "${YELLOW}Image URLs:${NC}"
echo -e "  ${FULL_IMAGE_NAME}"
echo -e "  ${LATEST_IMAGE_NAME}"
