#!/bin/bash
# Deploy BabyMilk to local k3d cluster
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DEPLOY_DIR="$(dirname "$PROJECT_DIR")/babymilk-deploy"
CHART_DIR="$DEPLOY_DIR/charts/babymilk"
NAMESPACE="babymilk"
RELEASE_NAME="babymilk"
TAG="${1:-latest}"

echo "=== BabyMilk Local Deployment ==="
echo ""

# Step 1: Build and push image
echo "[1/3] Building and pushing Docker image..."
"$SCRIPT_DIR/build-image.sh" "$TAG"
echo ""

# Step 2: Deploy with Helm
echo "[2/3] Deploying with Helm..."
helm upgrade --install "$RELEASE_NAME" "$CHART_DIR" \
    -f "$CHART_DIR/values-local.yaml" \
    -f "$CHART_DIR/values-local-secrets.yaml" \
    --set image.tag="$TAG" \
    -n "$NAMESPACE" --create-namespace \
    --wait --timeout 5m
echo ""

# Step 3: Wait and show status
echo "[3/3] Verifying deployment..."
kubectl get pods -n "$NAMESPACE"
echo ""
echo "=== Deployment complete ==="
echo "Access BabyMilk at: http://localhost:8080"
echo ""
echo "Useful commands:"
echo "  kubectl get all -n $NAMESPACE"
echo "  kubectl logs -n $NAMESPACE -l app.kubernetes.io/name=babymilk"
echo "  helm status $RELEASE_NAME -n $NAMESPACE"
