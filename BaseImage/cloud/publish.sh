#!/bin/bash

# Source environment variables
source .env

# Set variables
IMAGE_NAME="openai-morpheus-proxy"
REGISTRY="srt0422"

echo "Publishing version ${NFA_PROXY_VERSION} to Docker Hub..."
docker push ${REGISTRY}/${IMAGE_NAME}:${NFA_PROXY_VERSION}
docker push ${REGISTRY}/${IMAGE_NAME}:latest

echo "Publish completed successfully!" 