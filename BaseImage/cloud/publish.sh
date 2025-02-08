#!/bin/bash

# Source version from .env
VERSION=$(grep NFA_PROXY_VERSION ../.env | cut -d '"' -f2)

# Set variables
IMAGE_NAME="openai-morpheus-proxy"
REGISTRY="srt0422"  # Using Docker Hub registry

# Push to Docker Hub
echo "Pushing to Docker Hub..."
docker push ${REGISTRY}/${IMAGE_NAME}:${VERSION}
docker push ${REGISTRY}/${IMAGE_NAME}:latest

echo "Image published successfully to Docker Hub!" 