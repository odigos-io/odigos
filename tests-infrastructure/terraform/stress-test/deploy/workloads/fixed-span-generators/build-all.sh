#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Building and pushing all fixed span generators to ECR...${NC}"

# Build and push each language
echo -e "${YELLOW}Building Go fixed span generator...${NC}"
cd go && ./build-and-push.sh && cd ..

echo -e "${YELLOW}Building Python fixed span generator...${NC}"
cd python && ./build-and-push.sh && cd ..

echo -e "${YELLOW}Building Node.js fixed span generator...${NC}"
cd node && ./build-and-push.sh && cd ..

echo -e "${YELLOW}Building Java fixed span generator...${NC}"
cd java && ./build-and-push.sh && cd ..

echo -e "${GREEN}All fixed span generators built and pushed successfully!${NC}"
echo -e "${YELLOW}Images pushed:${NC}"
echo -e "  public.ecr.aws/odigos/go-fixed-span-gen:v1.0.0"
echo -e "  public.ecr.aws/odigos/python-fixed-span-gen:v1.0.0"
echo -e "  public.ecr.aws/odigos/node-fixed-span-gen:v1.0.0"
echo -e "  public.ecr.aws/odigos/java-fixed-span-gen:v1.0.0"
