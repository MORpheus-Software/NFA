#!/bin/zsh

# Enable verbose mode
set -x

MODEL_HANDLE="LMR-Hermes-2-Theta-Llama-3-8B"
# Use the PROVIDER_URL env var if set, otherwise default to localhost
PROVIDER_URL=${PROVIDER_URL:-"localhost:8081"}
# Use the SCHEME env var if set, otherwise default to http for local testing
SCHEME=${SCHEME:-"http"}
AUTH_HEADER="Basic $(echo -n 'admin:yosz9BZCuu7Rli7mYe4G1JbIO0Yprvwl' | base64)"

echo "=== Test Configuration ==="
echo "Using MODEL_HANDLE: $MODEL_HANDLE"
echo "Testing provider at: ${SCHEME}://${PROVIDER_URL}"
echo "Using auth header: $AUTH_HEADER"
echo "========================="

echo -e "\n=== Making chat completion request ==="
echo "Sending request to: ${SCHEME}://${PROVIDER_URL}/v1/chat/completions"
echo "With headers:"
echo "  Content-Type: application/json"
echo "  Authorization: $AUTH_HEADER"
echo "Request body:"
echo '{
    "model": "'"$MODEL_HANDLE"'",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
}'

curl -v -X POST "${SCHEME}://${PROVIDER_URL}/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -H "Authorization: ${AUTH_HEADER}" \
  -d '{
    "model": "'"$MODEL_HANDLE"'",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'

echo -e "\n=== Test Complete ==="