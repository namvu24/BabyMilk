#!/bin/bash
# Create k3d cluster with local Docker registry for MilkApp
set -e

REGISTRY_NAME="milkapp-registry"
REGISTRY_PORT=5111
CLUSTER_NAME="milkapp"

# Create registry if it doesn't exist
if k3d registry list | grep -q "$REGISTRY_NAME"; then
    echo "Registry '$REGISTRY_NAME' already exists, skipping..."
else
    echo "Creating local Docker registry..."
    k3d registry create "$REGISTRY_NAME" --port "$REGISTRY_PORT"
fi

# Create cluster if it doesn't exist
if k3d cluster list | grep -q "$CLUSTER_NAME"; then
    echo "Cluster '$CLUSTER_NAME' already exists, skipping..."
else
    echo "Creating k3d cluster..."
    k3d cluster create "$CLUSTER_NAME" \
        --registry-use "k3d-${REGISTRY_NAME}:${REGISTRY_PORT}" \
        -p "8080:80@loadbalancer" \
        --agents 1
fi

echo ""
echo "=== k3d environment ready ==="
echo "Cluster:  $CLUSTER_NAME"
echo "Registry: localhost:$REGISTRY_PORT (cluster: k3d-${REGISTRY_NAME}:${REGISTRY_PORT})"
echo "Ingress:  http://localhost:8080"
echo ""
echo "Next: run build-image script, then deploy-local script"
