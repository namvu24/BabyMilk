#!/bin/bash
# Deploy MilkApp to local k3d cluster
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DEPLOY_DIR="$(dirname "$PROJECT_DIR")/milkapp-deploy"
CHART_DIR="$DEPLOY_DIR/charts/milkapp"
NAMESPACE="milkapp"
RELEASE_NAME="milkapp"
TAG="${1:-latest}"

echo "=== MilkApp Local Deployment ==="
echo ""

# Step 1: Build and push image
echo "[1/4] Building and pushing Docker image..."
"$SCRIPT_DIR/build-image.sh" "$TAG"
echo ""

# Step 2: Add Bitnami repo and update dependencies
echo "[2/4] Updating Helm dependencies..."
helm repo add bitnami https://charts.bitnami.com/bitnami 2>/dev/null || true
helm repo update
helm dependency update "$CHART_DIR"
echo ""

# Step 3: Deploy with Helm
echo "[3/4] Deploying with Helm..."
helm upgrade --install "$RELEASE_NAME" "$CHART_DIR" \
    -f "$CHART_DIR/values-local.yaml" \
    --set image.tag="$TAG" \
    -n "$NAMESPACE" --create-namespace \
    --wait --timeout 5m
echo ""

# Step 4: Wait and show status
echo "[4/4] Verifying deployment..."
kubectl get pods -n "$NAMESPACE"
echo ""
echo "=== Deployment complete ==="
echo "Access MilkApp at: http://localhost:8080"
echo ""
echo "Useful commands:"
echo "  kubectl get all -n $NAMESPACE"
echo "  kubectl logs -n $NAMESPACE -l app.kubernetes.io/name=milkapp"
echo "  helm status $RELEASE_NAME -n $NAMESPACE"
