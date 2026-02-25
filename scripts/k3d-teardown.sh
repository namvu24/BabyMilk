#!/bin/bash
# Tear down k3d cluster and registry for MilkApp
set -e

CLUSTER_NAME="milkapp"
REGISTRY_NAME="milkapp-registry"
NAMESPACE="milkapp"

echo "=== Tearing down MilkApp k3d environment ==="

# Uninstall Helm release (if cluster is running)
if k3d cluster list 2>/dev/null | grep -q "$CLUSTER_NAME"; then
    echo "Uninstalling Helm release..."
    helm uninstall milkapp -n "$NAMESPACE" 2>/dev/null || true

    echo "Deleting k3d cluster..."
    k3d cluster delete "$CLUSTER_NAME"
else
    echo "Cluster '$CLUSTER_NAME' not found, skipping..."
fi

# Delete registry
if k3d registry list 2>/dev/null | grep -q "$REGISTRY_NAME"; then
    echo "Deleting Docker registry..."
    k3d registry delete "k3d-$REGISTRY_NAME"
else
    echo "Registry '$REGISTRY_NAME' not found, skipping..."
fi

echo ""
echo "=== Teardown complete ==="
