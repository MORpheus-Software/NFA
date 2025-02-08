#!/bin/bash

# Source version from .env
VERSION=$(grep NFA_PROXY_VERSION ../.env | cut -d '"' -f2)

# Set variables
IMAGE_NAME="openai-morpheus-proxy"
REGISTRY="srt0422"
TARGET_OS="linux"
TARGET_ARCH="amd64"
PLATFORM="${TARGET_OS}/${TARGET_ARCH}"

# Build the Docker image
echo "Building Docker image..."
docker build -f ../Dockerfile.proxy \
  --build-arg TARGETOS=${TARGET_OS} \
  --build-arg TARGETARCH=${TARGET_ARCH} \
  --platform ${PLATFORM} \
  -t ${REGISTRY}/${IMAGE_NAME}:${VERSION} -t ${REGISTRY}/${IMAGE_NAME}:latest ..

echo "Build completed successfully!"
