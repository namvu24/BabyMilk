#!/bin/bash
# Build and push MilkApp Docker image to Docker Hub
set -e

TAG="${1:-latest}"
IMAGE="namvu24/babymilk:$TAG"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Check Docker Hub login
if ! docker info 2>/dev/null | grep -q "Username"; then
    echo "Please log in to Docker Hub first:"
    docker login
fi

echo "Building Docker image: $IMAGE"
docker build -t "$IMAGE" "$PROJECT_DIR"

echo "Pushing to Docker Hub..."
docker push "$IMAGE"

echo ""
echo "=== Image pushed successfully ==="
echo "Image: $IMAGE"
