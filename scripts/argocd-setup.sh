#!/bin/bash
# Install ArgoCD on both homelab k3d clusters and apply Application manifests
# Run from main machine after kubeconfig-setup
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DEPLOY_DIR="$(dirname "$PROJECT_DIR")/babymilk-deploy"
ARGOCD_VERSION="v2.14.11"

echo "=== Installing ArgoCD on homelab clusters ==="
echo ""

for CLUSTER in babymilk-dev babymilk-prod; do
    CONTEXT="k3d-$CLUSTER"
    echo "--- [$CLUSTER] ---"
    echo ""

    # Install ArgoCD
    echo "[1/3] Installing ArgoCD $ARGOCD_VERSION..."
    kubectl --context "$CONTEXT" create namespace argocd 2>/dev/null || true
    kubectl --context "$CONTEXT" apply -n argocd \
        -f "https://raw.githubusercontent.com/argoproj/argo-cd/$ARGOCD_VERSION/manifests/install.yaml"
    echo ""

    # Wait for ArgoCD to be ready
    echo "[2/3] Waiting for ArgoCD to be ready..."
    kubectl --context "$CONTEXT" -n argocd rollout status deployment/argocd-server --timeout=120s
    echo ""

    # Apply the matching Application manifest
    echo "[3/3] Applying ArgoCD Application..."
    if [ "$CLUSTER" = "babymilk-dev" ]; then
        kubectl --context "$CONTEXT" apply -f "$DEPLOY_DIR/argocd/dev.yaml"
    else
        kubectl --context "$CONTEXT" apply -f "$DEPLOY_DIR/argocd/prod.yaml"
    fi
    echo ""

    # Print initial admin password
    echo "ArgoCD admin password for $CLUSTER:"
    kubectl --context "$CONTEXT" -n argocd get secret argocd-initial-admin-secret \
        -o jsonpath="{.data.password}" | base64 -d
    echo ""
    echo ""
done

echo "=== ArgoCD installed on both clusters ==="
echo ""
echo "Access ArgoCD UI (port-forward from main machine):"
echo "  Dev:  kubectl --context k3d-babymilk-dev port-forward svc/argocd-server -n argocd 9080:443"
echo "  Prod: kubectl --context k3d-babymilk-prod port-forward svc/argocd-server -n argocd 9081:443"
echo ""
echo "Then open: https://localhost:9080 (dev) or https://localhost:9081 (prod)"
echo "Login: admin / <password printed above>"
