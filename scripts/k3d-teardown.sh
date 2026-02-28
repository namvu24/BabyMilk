#!/bin/bash
# Tear down k3d cluster for BabyMilk
set -e

CLUSTER_NAME="babymilk"
NAMESPACE="babymilk"

echo "=== Tearing down BabyMilk k3d environment ==="

# Uninstall Helm release and delete cluster
if k3d cluster list 2>/dev/null | grep -q "$CLUSTER_NAME"; then
    echo "Uninstalling Helm release..."
    helm uninstall babymilk -n "$NAMESPACE" 2>/dev/null || true

    echo "Deleting k3d cluster..."
    k3d cluster delete "$CLUSTER_NAME"
else
    echo "Cluster '$CLUSTER_NAME' not found, skipping..."
fi

echo ""
echo "=== Teardown complete ==="
