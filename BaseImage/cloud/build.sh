#!/bin/bash

# Source environment variables
source .env

# Set variables
IMAGE_NAME="openai-morpheus-proxy"
REGISTRY="srt0422"
TARGET_OS="linux"
TARGET_ARCH="amd64"
PLATFORM="${TARGET_OS}/${TARGET_ARCH}"

# Build the Docker image
echo "Building Docker image with version ${NFA_PROXY_VERSION}..."
docker build -f ../Dockerfile.proxy \
  --build-arg TARGETOS=${TARGET_OS} \
  --build-arg TARGETARCH=${TARGET_ARCH} \
  --platform ${PLATFORM} \
  -t ${REGISTRY}/${IMAGE_NAME}:${NFA_PROXY_VERSION} -t ${REGISTRY}/${IMAGE_NAME}:latest ..

echo "Build completed successfully!"
