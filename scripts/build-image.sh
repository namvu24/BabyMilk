#!/bin/bash
# Build and push MilkApp Docker image to local registry
set -e

TAG="${1:-latest}"
REGISTRY="localhost:5111"
IMAGE="$REGISTRY/milkapp:$TAG"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Building Docker image: $IMAGE"
docker build -t "$IMAGE" "$PROJECT_DIR"

echo "Pushing to local registry..."
docker push "$IMAGE"

echo ""
echo "=== Image pushed successfully ==="
echo "Image: $IMAGE"
echo "Cluster ref: k3d-milkapp-registry:5111/milkapp:$TAG"
