#!/bin/bash

# Enable error handling and debug output
set -ex

# Source configuration if it exists
CONFIG_FILE="ExampleChatApp/cloud/config.sh"
if [ -f "$CONFIG_FILE" ]; then
    echo "Loading configuration from $CONFIG_FILE..."
    source "$CONFIG_FILE"
else
    echo "Warning: $CONFIG_FILE not found. Using default values..."
    
    # Default values from ExampleChatApp/cloud/config.sh
    export PROJECT_ID="vireo-401203"
    export REGION="us-west1"
    export ZONE="us-west1-a"
    export DOCKER_REGISTRY="srt0422"
    export CONSUMER_URL="https://consumer-node-2cmojdnxfq-uw.a.run.app"
    export MARKETPLACE_BASE_URL="${CONSUMER_URL}"
    export MARKETPLACE_URL="${MARKETPLACE_BASE_URL}"
    export DIAMOND_CONTRACT_ADDRESS="0xb8C55cD613af947E73E262F0d3C54b7211Af16CF"
    export MOR_TOKEN_ADDRESS="0x34a285a1b1c166420df5b6630132542923b5b27e"
    export CONSUMER_USERNAME="admin"
    export CONSUMER_PASSWORD="yosz9BZCuu7Rli7mYe4G1JbIO0Yprvwl"
    export BLOCKCHAIN_WS_URL="wss://arb-sepolia.g.alchemy.com/v2/HENS4s7bw4cBxIqyMtM-jryeMGZWj6du"
    export BLOCKCHAIN_HTTP_URL="https://arb-sepolia.g.alchemy.com/v2/HENS4s7bw4cBxIqyMtM-jryeMGZWj6du"
    export ETH_NODE_ADDRESS="${BLOCKCHAIN_WS_URL:-${BLOCKCHAIN_HTTP_URL:-https://sepolia-rollup.arbitrum.io/rpc}}"
fi

# Required environment variables for local development
# These match the variables set in deploy-proxy.sh
export INTERNAL_API_PORT="8081"
export MARKETPLACE_PORT="3333"
export MARKETPLACE_BASE_URL="${CONSUMER_URL}"
export MARKETPLACE_URL="${MARKETPLACE_BASE_URL}"
export CONSUMER_USERNAME="${CONSUMER_USERNAME}"
export CONSUMER_PASSWORD="${CONSUMER_PASSWORD}"
export CONSUMER_NODE_URL="${CONSUMER_URL}"
export SESSION_DURATION="1h"
export AUTH_TOKEN="${CONSUMER_PASSWORD}"

# Additional environment variables from config.sh
export DIAMOND_CONTRACT_ADDRESS="${DIAMOND_CONTRACT_ADDRESS}"
export MOR_TOKEN_ADDRESS="${MOR_TOKEN_ADDRESS}"
export ETH_NODE_ADDRESS="${ETH_NODE_ADDRESS}"
export ETH_NODE_LEGACY_TX="false"
export ETH_NODE_USE_SUBSCRIPTIONS="true"
export ETH_NODE_CHAIN_ID="421614"
export ENVIRONMENT="development"
export LOG_LEVEL="debug"
export LOG_FORMAT="text"
export LOG_COLOR="true"
export PROXY_STORE_CHAT_CONTEXT="true"
export PROXY_STORAGE_PATH="/tmp"
export PROVIDER_CACHE_TTL="60"
export MAX_CONCURRENT_SESSIONS="100"
export SESSION_TIMEOUT="3600"
export EXPLORER_API_URL="https://api-sepolia.arbiscan.io/api"

echo "=== Environment Configuration ==="
echo "MARKETPLACE_URL: $MARKETPLACE_URL"
echo "CONSUMER_NODE_URL: $CONSUMER_NODE_URL"
echo "INTERNAL_API_PORT: $INTERNAL_API_PORT"
echo "MARKETPLACE_PORT: $MARKETPLACE_PORT"
echo "AUTH_TOKEN: $AUTH_TOKEN"
echo "LOG_LEVEL: $LOG_LEVEL"
echo "========================="

# Change to BaseImage directory and run the service
cd "$(dirname "$0")/../BaseImage"
echo "Starting BaseImage service..."

# Run in background with nohup
nohup go run main.go > proxy.log 2>&1 &

# Wait a moment for the service to start
sleep 2

# Check if service is running
if ! lsof -i :8081 > /dev/null; then
    echo "Error: Service failed to start. Check proxy.log for details"
    exit 1
fi

echo "Service started successfully on port 8081"
echo "Logs available in BaseImage/proxy.log" 