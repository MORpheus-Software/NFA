#!/bin/zsh

# Source environment variables
set -a
source .env
set +a

# Build configuration
BUILD_PLATFORM=${BUILD_PLATFORM:-"linux/amd64"}

# Repository management
REPO_PATH="./Morpheus-Lumerin-Node"
REPO_URL="https://github.com/MorpheusAIs/Morpheus-Lumerin-Node.git"

# Check and setup repository
if [ ! -d "$REPO_PATH" ]; then
    echo "Cloning Lumerin proxy-router repository..."
    git clone $REPO_URL $REPO_PATH
else
    cd $REPO_PATH
    
    # Check for local changes
    if [ -n "$(git status --porcelain)" ]; then
        echo "Local changes detected in proxy-router..."
        # Pull latest changes if there are local modifications
        git pull
        if [ $? -ne 0 ]; then
            echo "Error: Failed to pull latest changes. Please resolve conflicts first."
            exit 1
        fi
    fi
    cd ..
fi

echo "Building provider image for ${BUILD_PLATFORM}..."
docker buildx create --use --name marketplace-builder --platform ${BUILD_PLATFORM} || true
docker buildx build --platform ${BUILD_PLATFORM} \
  -t $REGISTRY/$PROVIDER_IMAGE_NAME:$VERSION \
  -t $REGISTRY/$PROVIDER_IMAGE_NAME:latest \
  --build-arg WALLET_ADDRESS=$PROVIDER_WALLET_ADDRESS \
  --load \
  -f $REPO_PATH/proxy-router/Dockerfile \
  $REPO_PATH/proxy-router

echo "Building consumer image for ${BUILD_PLATFORM}..."
docker buildx build --platform ${BUILD_PLATFORM} \
  -t $REGISTRY/$CONSUMER_IMAGE_NAME:$VERSION \
  -t $REGISTRY/$CONSUMER_IMAGE_NAME:latest \
  --build-arg WALLET_ADDRESS=$CONSUMER_WALLET_ADDRESS \
  --load \
  -f $REPO_PATH/proxy-router/Dockerfile \
  $REPO_PATH/proxy-router

# Retag images manually to ensure the 'latest' tag exists
docker tag $REGISTRY/$PROVIDER_IMAGE_NAME:$VERSION $REGISTRY/$PROVIDER_IMAGE_NAME:latest
docker tag $REGISTRY/$CONSUMER_IMAGE_NAME:$VERSION $REGISTRY/$CONSUMER_IMAGE_NAME:latest

echo "All images built successfully!"
