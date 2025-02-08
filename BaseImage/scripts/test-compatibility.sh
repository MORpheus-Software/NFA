#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Test configuration
API_PORT=${PORT:-8080}
BASE_URL="http://localhost:${API_PORT}"
AUTH_TOKEN="test-auth-token-12345"

# Helper function for logging
log_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

log_error() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}

# Wait for service to be healthy
wait_for_health() {
    echo "Waiting for service to be healthy..."
    for i in {1..30}; do
        if curl -s "${BASE_URL}/health" | grep -q "healthy"; then
            log_success "Service is healthy"
            return 0
        fi
        sleep 1
    done
    log_error "Service failed to become healthy"
}

# Test health endpoint
test_health() {
    echo "Testing health endpoint..."
    response=$(curl -s "${BASE_URL}/health")
    if echo "$response" | grep -q "healthy"; then
        log_success "Health check passed"
    else
        log_error "Health check failed: $response"
    fi
}

# Test model listing
test_models() {
    echo "Testing model listing..."
    response=$(curl -s -H "Authorization: Bearer ${AUTH_TOKEN}" "${BASE_URL}/blockchain/models")
    if echo "$response" | grep -q "models"; then
        log_success "Model listing successful"
    else
        log_error "Model listing failed: $response"
    fi
}

# Test chat completion with Bearer token
test_chat_completion() {
    echo "Testing chat completion with Bearer token..."
    response=$(curl -s -X POST \
        -H "Authorization: Bearer ${AUTH_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Hello"}],
            "stream": false
        }' \
        "${BASE_URL}/v1/chat/completions")
    
    if echo "$response" | grep -q "choices"; then
        log_success "Chat completion successful"
    else
        log_error "Chat completion failed: $response"
    fi
}

# Test streaming chat completion
test_streaming_chat() {
    echo "Testing streaming chat completion..."
    response=$(curl -s -N -X POST \
        -H "Authorization: Bearer ${AUTH_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Hello"}],
            "stream": true
        }' \
        "${BASE_URL}/v1/chat/completions")
    
    if echo "$response" | grep -q "data:"; then
        log_success "Streaming chat completion successful"
    else
        log_error "Streaming chat completion failed: $response"
    fi
}

# Test invalid model name
test_invalid_model() {
    echo "Testing invalid model name..."
    response=$(curl -s -X POST \
        -H "Authorization: Bearer ${AUTH_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "invalid-model-name",
            "messages": [{"role": "user", "content": "Hello"}]
        }' \
        "${BASE_URL}/v1/chat/completions")
    
    if echo "$response" | grep -q "No matching model found"; then
        log_success "Invalid model handling successful"
    else
        log_error "Invalid model handling failed: $response"
    fi
}

# Test missing auth token
test_missing_auth() {
    echo "Testing missing authentication..."
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{
            "model": "gpt-4",
            "messages": [{"role": "user", "content": "Hello"}]
        }' \
        "${BASE_URL}/v1/chat/completions")
    
    if echo "$response" | grep -q "Missing session token"; then
        log_success "Missing auth handling successful"
    else
        log_error "Missing auth handling failed: $response"
    fi
}

# Main test execution
main() {
    echo "Starting compatibility tests..."
    
    # Wait for service to be ready
    wait_for_health
    
    # Run tests
    test_health
    test_models
    test_chat_completion
    test_streaming_chat
    test_invalid_model
    test_missing_auth
    
    echo -e "\n${GREEN}All compatibility tests passed!${NC}"
}

# Run main function
main 