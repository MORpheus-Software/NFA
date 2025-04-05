#!/bin/bash
set -e

# Source environment variables
set -a
source ./.env
set +a

# Delete previous deployment and configmaps
echo "Deleting old deployment and resources..."
kubectl delete deployment provider-deployment || true
kubectl delete configmap cookie-template || true
kubectl delete configmap dynamic-web-url || true
kubectl delete configmap rating-config || true
sleep 5

# Create cookie ConfigMap
echo "Creating cookie ConfigMap with auth credentials..."
kubectl create configmap cookie-template --from-literal=.cookie="${PROVIDER_AUTH_USERNAME}:${PROVIDER_AUTH_PASSWORD}"

# Create web URL ConfigMap
echo "Creating dynamic web URL ConfigMap..."
kubectl create configmap dynamic-web-url --from-literal=url="http://provider-service:${PROVIDER_HEALTH_PORT}"

# Create a temporary deployment file with substituted variables
echo "Creating deployment with properly substituted variables..."
envsubst < provider-deployment.yaml > provider-deployment-temp.yaml
kubectl apply -f provider-deployment-temp.yaml
rm provider-deployment-temp.yaml

# Wait for the pod to start
echo "Waiting for pod to start..."
kubectl wait --for=condition=ready pod -l app=provider --timeout=60s || true

# Check logs for debug
echo "Done. Check logs with: kubectl logs -f deployment/provider-deployment"
echo "You can also check the debug container with: kubectl exec -it \$(kubectl get pod -l app=provider -o jsonpath='{.items[0].metadata.name}') -c debug -- sh"
echo "To check auth file: kubectl exec -it \$(kubectl get pod -l app=provider -o jsonpath='{.items[0].metadata.name}') -c debug -- cat /app/config/proxy.conf" 