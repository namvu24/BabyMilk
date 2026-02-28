#!/bin/bash
# Create k3d cluster for BabyMilk
set -e

CLUSTER_NAME="babymilk"

# Create cluster if it doesn't exist
if k3d cluster list | grep -q "$CLUSTER_NAME"; then
    echo "Cluster '$CLUSTER_NAME' already exists, skipping..."
else
    echo "Creating k3d cluster..."
    k3d cluster create "$CLUSTER_NAME" \
        -p "8080:80@loadbalancer" \
        --agents 1
fi

echo ""
echo "=== k3d environment ready ==="
echo "Cluster:  $CLUSTER_NAME"
echo "Ingress:  http://localhost:8080"
echo ""
echo "Next: run build-image script, then deploy-local script"
