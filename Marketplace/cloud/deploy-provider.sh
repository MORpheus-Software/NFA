#!/bin/bash
set -e

# Script for deploying or updating the Lumerin provider node in Kubernetes
# Usage: ./deploy-provider.sh [--skip-cleanup] [--custom-web-url=URL]

SCRIPT_DIR="$(dirname "$0")"
SKIP_CLEANUP=false
CUSTOM_WEB_URL=""

# Parse command line arguments
for arg in "$@"; do
  case $arg in
    --skip-cleanup)
      SKIP_CLEANUP=true
      shift
      ;;
    --custom-web-url=*)
      CUSTOM_WEB_URL="${arg#*=}"
      shift
      ;;
  esac
done

# Function to print colored messages
print_status() {
  local color="\033[1;36m"  # Cyan
  local reset="\033[0m"
  echo -e "${color}===> $1${reset}"
}

print_success() {
  local color="\033[1;32m"  # Green
  local reset="\033[0m"
  echo -e "${color}✓ $1${reset}"
}

print_warning() {
  local color="\033[1;33m"  # Yellow
  local reset="\033[0m"
  echo -e "${color}⚠ $1${reset}"
}

print_error() {
  local color="\033[1;31m"  # Red
  local reset="\033[0m"
  echo -e "${color}✗ $1${reset}"
  exit 1
}

# Source environment variables
print_status "Loading environment variables from .env"
if [ ! -f "${SCRIPT_DIR}/.env" ]; then
  print_error "Error: .env file not found in ${SCRIPT_DIR}"
fi

set -a
source "${SCRIPT_DIR}/.env"
set +a

# Check required environment variables
check_env_var() {
  local var_name=$1
  if [ -z "${!var_name}" ]; then
    print_error "Error: ${var_name} environment variable is required but not set in .env"
  fi
}

required_vars=(
  "PROVIDER_WALLET_PRIVATE_KEY"
  "DIAMOND_CONTRACT_ADDRESS"
  "MOR_TOKEN_ADDRESS"
  "ETH_NODE_ADDRESS"
  "ETH_NODE_CHAIN_ID"
  "PROVIDER_AUTH_USERNAME"
  "PROVIDER_AUTH_PASSWORD"
  "PROVIDER_PORT"
  "PROVIDER_HEALTH_PORT"
)

for var in "${required_vars[@]}"; do
  check_env_var "$var"
done

print_success "Environment variables loaded successfully"

# Ensure provider-secrets exists or create it
print_status "Setting up provider-secrets..."
if kubectl get secret provider-secrets &>/dev/null; then
  print_status "Updating provider-secrets with wallet private key"
  kubectl delete secret provider-secrets
else
  print_status "Creating provider-secrets with wallet private key"
fi

kubectl create secret generic provider-secrets --from-literal=wallet-private-key="${PROVIDER_WALLET_PRIVATE_KEY}"
print_success "Provider secrets configured"

# Clean up previous resources if not skipped
if [ "$SKIP_CLEANUP" = false ]; then
  print_status "Cleaning up previous deployment and resources..."
  kubectl delete deployment provider-deployment 2>/dev/null || true
  kubectl delete configmap dynamic-web-url 2>/dev/null || true
  kubectl delete configmap rating-config 2>/dev/null || true
  print_success "Previous resources cleaned up"
  sleep 3
else
  print_warning "Skipping cleanup of previous resources"
fi

# Create web URL ConfigMap
print_status "Creating dynamic web URL ConfigMap..."
kubectl delete configmap dynamic-web-url 2>/dev/null || true
if [ -n "$CUSTOM_WEB_URL" ]; then
  print_status "Using custom web URL: $CUSTOM_WEB_URL"
  WEB_URL="$CUSTOM_WEB_URL"
else
  # Check if a service exists and has an external IP
  if kubectl get service provider-service &>/dev/null; then
    SERVICE_IP=$(kubectl get service provider-service -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
    if [ -n "$SERVICE_IP" ]; then
      print_status "Using service external IP: $SERVICE_IP"
      WEB_URL="http://${SERVICE_IP}:${PROVIDER_HEALTH_PORT}"
      
      # Update service with proper environment variables if needed
      print_status "Applying provider-service with proper port settings..."
      envsubst < "${SCRIPT_DIR}/provider-service.yaml" > "${SCRIPT_DIR}/provider-service-temp.yaml"
      kubectl apply -f "${SCRIPT_DIR}/provider-service-temp.yaml"
      rm "${SCRIPT_DIR}/provider-service-temp.yaml"
      
      # Force service to finalize by describing it
      kubectl describe service provider-service > /dev/null
    else
      print_warning "No external IP found for provider-service, using default internal URL"
      WEB_URL="http://provider-service:${PROVIDER_HEALTH_PORT}"
      
      # Try to create the service if it doesn't have an IP
      print_status "Deploying provider-service to acquire an external IP..."
      envsubst < "${SCRIPT_DIR}/provider-service.yaml" > "${SCRIPT_DIR}/provider-service-temp.yaml"
      kubectl apply -f "${SCRIPT_DIR}/provider-service-temp.yaml"
      rm "${SCRIPT_DIR}/provider-service-temp.yaml"
      
      # Wait briefly for IP assignment
      print_status "Waiting for external IP assignment..."
      for i in {1..5}; do
        sleep 5
        SERVICE_IP=$(kubectl get service provider-service -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
        if [ -n "$SERVICE_IP" ]; then
          print_success "External IP assigned: $SERVICE_IP"
          WEB_URL="http://${SERVICE_IP}:${PROVIDER_HEALTH_PORT}"
          break
        fi
        echo "Waiting for IP assignment... (attempt $i/5)"
      done
    fi
  else
    print_warning "provider-service not found, creating it now"
    # Create the service to get an external IP
    envsubst < "${SCRIPT_DIR}/provider-service.yaml" > "${SCRIPT_DIR}/provider-service-temp.yaml"
    kubectl apply -f "${SCRIPT_DIR}/provider-service-temp.yaml"
    rm "${SCRIPT_DIR}/provider-service-temp.yaml"
    
    # Use the default internal URL for now
    WEB_URL="http://provider-service:${PROVIDER_HEALTH_PORT}"
    
    # Wait briefly for IP assignment
    print_status "Waiting for external IP assignment..."
    for i in {1..5}; do
      sleep 5
      SERVICE_IP=$(kubectl get service provider-service -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
      if [ -n "$SERVICE_IP" ]; then
        print_success "External IP assigned: $SERVICE_IP"
        WEB_URL="http://${SERVICE_IP}:${PROVIDER_HEALTH_PORT}"
        break
      fi
      echo "Waiting for IP assignment... (attempt $i/5)"
    done
  fi
fi

kubectl create configmap dynamic-web-url --from-literal=url="${WEB_URL}"
print_success "Web URL ConfigMap created with URL: ${WEB_URL}"

# Apply models ConfigMap if needed
if [ ! -z "$MODEL_ID" ] && [ ! -z "$MODEL_API_URL" ] && [ ! -z "$MODEL_API_KEY" ]; then
  print_status "Configuring model settings..."
  # Check if models-config exists or update it
  if kubectl get configmap models-config &>/dev/null; then
    kubectl delete configmap models-config
  fi
  
  # Create a temporary models-config.json file
  cat > models-config-temp.json << EOF
{
  "\$schema": "./internal/config/models-config-schema.json",
  "models": [
    {
      "modelId": "${MODEL_ID}",
      "modelName": "${MODEL_NAME:-Default Model}",
      "apiType": "${MODEL_API_TYPE:-openai}",
      "apiUrl": "${MODEL_API_URL}",
      "apiKey": "${MODEL_API_KEY}"
    }
  ]
}
EOF

  kubectl create configmap models-config --from-file=models-config.json=models-config-temp.json
  rm models-config-temp.json
  print_success "Model configuration applied"
fi

# Generate the salt and hash for authentication
# This keeps the original password in the .env file but uses the proper format in proxy.conf
print_status "Generating auth salt and hash..."
SALT=$(python3 -c "import secrets; print(secrets.token_hex(16))")
HASH=$(python3 -c "import hmac; password='${PROVIDER_AUTH_PASSWORD}'; salt='${SALT}'; m = hmac.new(salt.encode('utf-8'), password.encode('utf-8'), 'SHA256'); print(m.hexdigest())")

# Export these for use in the deployment
export PROVIDER_AUTH_SALT=$SALT
export PROVIDER_AUTH_HASH=$HASH

print_status "Auth hash generated for ${PROVIDER_AUTH_USERNAME}"

# Create a temporary deployment file with substituted variables
print_status "Creating deployment with properly substituted variables..."
if [ ! -f "${SCRIPT_DIR}/provider-deployment.yaml" ]; then
  print_error "Error: provider-deployment.yaml not found in ${SCRIPT_DIR}"
fi

# Create a temporary file with proper variable expansion
cp "${SCRIPT_DIR}/provider-deployment.yaml" "${SCRIPT_DIR}/provider-deployment-temp.yaml"

# Manually substitute environment variables to avoid issues with envsubst
sed -i "" "s|\${PROVIDER_PORT}|${PROVIDER_PORT}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${PROVIDER_HEALTH_PORT}|${PROVIDER_HEALTH_PORT}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${REGISTRY}|${REGISTRY}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${PROVIDER_IMAGE_NAME}|${PROVIDER_IMAGE_NAME}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${VERSION}|${VERSION}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${DIAMOND_CONTRACT_ADDRESS}|${DIAMOND_CONTRACT_ADDRESS}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${MOR_TOKEN_ADDRESS}|${MOR_TOKEN_ADDRESS}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${EXPLORER_API_URL}|${EXPLORER_API_URL}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${ETH_NODE_CHAIN_ID}|${ETH_NODE_CHAIN_ID}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${ENVIRONMENT}|${ENVIRONMENT}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${ETH_NODE_ADDRESS}|${ETH_NODE_ADDRESS}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${MODELS_CONFIG_PATH}|${MODELS_CONFIG_PATH}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${ETH_NODE_USE_SUBSCRIPTIONS}|${ETH_NODE_USE_SUBSCRIPTIONS}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${ETH_NODE_LEGACY_TX}|${ETH_NODE_LEGACY_TX}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${PROXY_STORAGE_PATH}|${PROXY_STORAGE_PATH}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${PROVIDER_AUTH_USERNAME}|${PROVIDER_AUTH_USERNAME}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${PROVIDER_AUTH_PASSWORD}|${PROVIDER_AUTH_PASSWORD}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${PROVIDER_AUTH_SALT}|${PROVIDER_AUTH_SALT}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"
sed -i "" "s|\${PROVIDER_AUTH_HASH}|${PROVIDER_AUTH_HASH}|g" "${SCRIPT_DIR}/provider-deployment-temp.yaml"

kubectl apply -f "${SCRIPT_DIR}/provider-deployment-temp.yaml"
rm "${SCRIPT_DIR}/provider-deployment-temp.yaml"

# Wait for the pod to start
print_status "Waiting for pod to start..."
kubectl wait --for=condition=ready pod -l app=provider --timeout=120s || true

# Check deployment status
if kubectl get pods -l app=provider | grep -q "Running"; then
  print_success "Provider deployment is running!"
  
  # Get the pod name for further commands
  POD_NAME=$(kubectl get pod -l app=provider -o jsonpath='{.items[0].metadata.name}')
  
  # Print useful information
  echo ""
  print_status "DEPLOYMENT SUMMARY"
  echo "Provider Pod: $POD_NAME"
  echo "Wallet Address: Derived from private key in provider-secrets"
  echo "Environment: ${ENVIRONMENT}"
  echo "Ethereum Node: ${ETH_NODE_ADDRESS}"
  echo "Diamond Contract: ${DIAMOND_CONTRACT_ADDRESS}"
  echo "Web URL: ${WEB_URL}"
  echo ""
  
  # Verify the WEB_PUBLIC_URL is correctly set
  print_status "Verifying deployment configuration..."
  kubectl exec -it $POD_NAME -c debug -- sh -c "echo 'Web Public URL (from Kubernetes ConfigMap):' && cat /proc/1/environ | tr '\0' '\n' | grep WEB_PUBLIC_URL" 2>/dev/null || print_warning "Could not verify WEB_PUBLIC_URL in the container"
  
  print_status "HELPFUL COMMANDS"
  echo "Check logs: kubectl logs -f deployment/provider-deployment"
  echo "Access debug container: kubectl exec -it ${POD_NAME} -c debug -- sh"
  echo "Port forward to provider: kubectl port-forward service/provider-service ${PROVIDER_PORT}:${PROVIDER_PORT}"
  echo ""
  
  print_status "To update the WEB_PUBLIC_URL after deployment:"
  echo "  ./deploy-provider.sh --skip-cleanup --custom-web-url=http://your-custom-url:${PROVIDER_HEALTH_PORT}"
  echo ""
else
  print_error "Provider deployment failed to start properly"
fi
