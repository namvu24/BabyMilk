#!/bin/bash
# Create dev and prod k3d clusters on the homelab laptop
# Run from main machine — connects via SSH to nam@homelab
set -e

REMOTE="nam@homelab"
REMOTE_HOST="homelab"
REMOTE_IP="192.168.1.63"

echo "=== Setting up k3d clusters on $REMOTE ==="
echo ""

# Create dev cluster
echo "[1/2] Creating babymilk-dev cluster..."
ssh "$REMOTE" bash -s <<'REMOTE_SCRIPT'
set -e

# Dev cluster: API on 6444, ingress on 8081
if k3d cluster list 2>/dev/null | grep -q "babymilk-dev"; then
    echo "Cluster 'babymilk-dev' already exists, skipping..."
else
    k3d cluster create babymilk-dev \
        --api-port 0.0.0.0:6444 \
        -p "8081:80@loadbalancer" \
        --agents 1 \
        --k3s-arg "--tls-san=homelab@server:*" \
        --k3s-arg "--tls-san=192.168.1.63@server:*"
    echo "babymilk-dev cluster created"
fi
REMOTE_SCRIPT

echo ""

# Create prod cluster
echo "[2/2] Creating babymilk-prod cluster..."
ssh "$REMOTE" bash -s <<'REMOTE_SCRIPT'
set -e

# Prod cluster: API on 6445, ingress on 8082
if k3d cluster list 2>/dev/null | grep -q "babymilk-prod"; then
    echo "Cluster 'babymilk-prod' already exists, skipping..."
else
    k3d cluster create babymilk-prod \
        --api-port 0.0.0.0:6445 \
        -p "8082:80@loadbalancer" \
        --agents 1 \
        --k3s-arg "--tls-san=homelab@server:*" \
        --k3s-arg "--tls-san=192.168.1.63@server:*"
    echo "babymilk-prod cluster created"
fi
REMOTE_SCRIPT

echo ""
echo "=== Both clusters created on $REMOTE_HOST ==="
echo ""
echo "Next: run kubeconfig-setup to merge remote kubeconfigs"
