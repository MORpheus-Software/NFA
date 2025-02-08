#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROXY_URL="https://nfa-proxy-874161723906.us-west1.run.app"
AUTH_USER="admin"
AUTH_PASS="yosz9BZCuu7Rli7mYe4G1JbIO0Yprvwl"

# Arrays of model IDs and names (parallel arrays)
MODEL_IDS=(
    "0x560d9704d2dba7da8dab2db043f9f8fd9354936561961569ddd874641adee13e"
    # "0xd8a304f87466ed3f8da0195fc67ba71cdca881100b5ad7855acd211d09966722"
    # "0xa2bdc7ce113aeaebafe9a6e26ef8fc86cdf0328ea21ba0748944ea74e9539431"
)

MODEL_NAMES=(
    "LMR-Hermes-2-Theta-Llama-3-8B"
    # "LMR-Capybara Hermes 2.5 Mistral-7B"
    # "LMR-HyperB-Qwen2.5-Coder-32B"
)

# Function to check container health
check_health() {
    echo -e "${YELLOW}Checking proxy health...${NC}"
    response=$(curl -s -o /dev/null -w "%{http_code}" ${PROXY_URL}/health)
    if [ "$response" = "200" ]; then
        echo -e "${GREEN}Proxy is healthy${NC}"
        return 0
    fi
    echo -e "${RED}Proxy is not healthy${NC}"
    return 1
}

# Function to get wallet address
get_wallet_address() {
    wallet_response=$(curl -s -X GET "${PROXY_URL}/blockchain/wallet" -H 'accept: application/json')
    if [[ $wallet_response == *"address"* ]]; then
        echo $(echo $wallet_response | jq -r '.address')
    else
        echo ""
    fi
}

# Function to cleanup all sessions
cleanup_sessions() {
    echo -e "\n${YELLOW}Cleaning up existing sessions...${NC}"
    wallet_address=$(get_wallet_address)
    
    if [ -z "$wallet_address" ]; then
        echo -e "${YELLOW}No wallet address found, skipping session cleanup${NC}"
        return
    fi
    
    # Get all sessions for the user
    sessions_response=$(curl -s -X GET "${PROXY_URL}/blockchain/sessions/user?user=${wallet_address}" \
        -H 'accept: application/json')
    
    # Extract session IDs and close each session
    if [[ $sessions_response == *"sessions"* ]]; then
        echo $sessions_response | jq -r '.sessions[]?.id' | while read -r session_id; do
            if [ ! -z "$session_id" ]; then
                echo -e "${YELLOW}Closing session: $session_id${NC}"
                close_response=$(curl -s -X POST "${PROXY_URL}/blockchain/sessions/${session_id}/close" \
                    -H 'accept: application/json')
                echo -e "${GREEN}Close response: $close_response${NC}"
            fi
        done
    else
        echo -e "${YELLOW}No active sessions found${NC}"
    fi
}

# Function to send chat request
send_chat_request() {
    local session_id="$1"
    local model_name="$2"
    local message="$3"
    local step_name="$4"
    
    echo -e "\n${YELLOW}$step_name${NC}"
    echo -e "Using model: ${GREEN}$model_name${NC}"
    echo -e "Using session: ${GREEN}$session_id${NC}"
    
    chat_response=$(curl -s -X POST "${PROXY_URL}/v1/chat/completions" \
        -H 'accept: application/json' \
        -H 'Content-Type: application/json' \
        -H "session_id: $session_id" \
        -H "Authorization: Basic $(echo -n "${AUTH_USER}:${AUTH_PASS}" | base64)" \
        -d "{
            \"messages\": [{\"content\": \"$message\", \"role\": \"user\"}],
            \"stream\": false,
            \"model\": \"$model_name\"
        }")

    echo -e "${GREEN}Chat Response:${NC}"
    echo "$chat_response"
    
    return 0
}

# Check health first
check_health
if [ $? -ne 0 ]; then
    echo -e "${RED}Proxy health check failed. Please ensure the proxy is running.${NC}"
    exit 1
fi

# Clean up existing sessions before starting
cleanup_sessions

# Main testing loop
for i in "${!MODEL_IDS[@]}"; do
    model_id="${MODEL_IDS[$i]}"
    model_name="${MODEL_NAMES[$i]}"
    echo -e "\n${YELLOW}Testing model: $model_name${NC}"
    echo -e "Model ID: $model_id"
    
    # Create session
    echo -e "\n${YELLOW}Creating session for $model_name${NC}"
    session_response=$(curl -s -X POST "${PROXY_URL}/blockchain/models/${model_id}/session" \
        -H 'accept: application/json' \
        -H 'Content-Type: application/json' \
        -H "Authorization: Basic $(echo -n "${AUTH_USER}:${AUTH_PASS}" | base64)" \
        -d '{"sessionDuration": 600}')
    
    # Check if response contains error but continue anyway
    if [[ $session_response == *"error"* ]] || [[ $session_response == *"Failed"* ]]; then
        if [[ $session_response == *"ERC20: transfer amount exceeds balance"* ]] || [[ $session_response == *"insufficient funds"* ]]; then
            echo -e "${YELLOW}Note: Model needs tokens but continuing test: $model_name${NC}"
        elif [[ $session_response == *"no provider accepting session"* ]]; then
            echo -e "${YELLOW}Note: No providers currently available for: $model_name${NC}"
        else
            echo -e "${YELLOW}Session creation warning: $session_response${NC}"
        fi
    fi
    
    # Extract session ID
    session_id=$(echo $session_response | jq -r '.sessionID')
    if [ -z "$session_id" ] || [ "$session_id" = "null" ]; then
        echo -e "${RED}Could not get session ID for $model_name. Response: $session_response${NC}"
        continue
    fi
    
    echo -e "${GREEN}Created session: $session_id${NC}"
    
    # Test chat request immediately
    send_chat_request "$session_id" "$model_name" "Hello, can you help me test if you're working?" "Testing chat with $model_name"
    
    # Close session
    echo -e "\n${YELLOW}Closing session for $model_name${NC}"
    close_response=$(curl -s -X POST "${PROXY_URL}/blockchain/sessions/${session_id}/close" \
        -H 'accept: application/json')
    
    if [[ $close_response == *"404"* ]]; then
        echo -e "${YELLOW}Note: Session may have already been closed or expired${NC}"
    else
        echo -e "${GREEN}Session closed successfully${NC}"
    fi
    
    # Add a small delay between tests
    sleep 2
done 