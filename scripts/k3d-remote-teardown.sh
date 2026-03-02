#!/bin/bash
# Tear down dev and prod k3d clusters on the homelab laptop
# Run from main machine
set -e

REMOTE="nam@homelab"
REMOTE_HOST="homelab"

echo "=== Tearing down k3d clusters on $REMOTE_HOST ==="
echo ""

for CLUSTER in babymilk-dev babymilk-prod; do
    echo "Deleting $CLUSTER..."
    ssh "$REMOTE" "k3d cluster delete $CLUSTER 2>/dev/null || true"

    # Remove context from local kubeconfig
    kubectl config delete-context "k3d-$CLUSTER" 2>/dev/null || true
    kubectl config delete-cluster "k3d-$CLUSTER" 2>/dev/null || true
    kubectl config delete-user "admin@k3d-$CLUSTER" 2>/dev/null || true
done

echo ""
echo "=== Teardown complete ==="
