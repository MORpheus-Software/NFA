#!/bin/zsh

# Source environment variables
set -a
source .env
set +a

echo "Creating provider secrets..."
kubectl create secret generic provider-secrets \
    --from-literal=wallet-private-key=$PROVIDER_WALLET_PRIVATE_KEY \
    --dry-run=client -o yaml | kubectl apply -f -

echo "Creating provider PVC..."
kubectl apply -f provider-pvc.yaml

# Create an empty ConfigMap for the dynamic web URL
echo "Creating dynamic web URL ConfigMap..."
kubectl create configmap dynamic-web-url --from-literal=url="" --dry-run=client -o yaml | kubectl apply -f -

echo "Deploying provider..."
envsubst < provider-deployment.yaml | kubectl apply -f -

echo "Waiting for provider service to get an external IP..."
kubectl wait --for=condition=available deployment/provider-deployment --timeout=180s
