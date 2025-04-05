#!/bin/bash
set -e

# Delete previous deployment and configmaps
echo "Deleting old deployment and resources..."
kubectl delete deployment provider-deployment || true
kubectl delete configmap proxy-config || true
kubectl delete configmap cookie-template || true
kubectl delete configmap dynamic-web-url || true
sleep 5

# Create proper proxy config 
echo "Creating proxy-config ConfigMap with proper auth format..."
kubectl create configmap proxy-config --from-literal=proxy.conf="rpcauth=admin:87d7ac3c47e29f6568f19aee24f\$e02287316968123246aa07a0c67e3d09f69891b53f26422cf49cd5c05cd0bd
rpcwhitelist=admin:*
rpcwhitelistdefault=0"

# Create cookie ConfigMap
echo "Creating cookie ConfigMap with auth credentials..."
kubectl create configmap cookie-template --from-literal=.cookie="admin:JJLRNze08ZN3vlNdgwgbrh6c4dRw9gQT"

# Create web URL ConfigMap
echo "Creating dynamic web URL ConfigMap..."
kubectl create configmap dynamic-web-url --from-literal=url="http://provider-service:8082"

# Deploy again
echo "Redeploying provider..."
kubectl apply -f provider-deployment.yaml

echo "Done. Check logs with: kubectl logs -f deployment/provider-deployment" 