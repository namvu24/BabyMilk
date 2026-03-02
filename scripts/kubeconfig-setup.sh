#!/bin/bash
# Fetch kubeconfigs from homelab k3d clusters and merge into local ~/.kube/config
# Run from main machine after k3d-remote-setup.sh
set -e

REMOTE="nam@homelab"
REMOTE_HOST="homelab"

echo "=== Fetching kubeconfigs from $REMOTE_HOST ==="
echo ""

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

for CLUSTER in babymilk-dev babymilk-prod; do
    echo "Fetching kubeconfig for $CLUSTER..."

    # Get kubeconfig from remote
    ssh "$REMOTE" "k3d kubeconfig get $CLUSTER" > "$TMPDIR/$CLUSTER.yaml"

    # Replace localhost/0.0.0.0 with homelab hostname
    sed -i "s|https://0\.0\.0\.0:|https://$REMOTE_HOST:|g" "$TMPDIR/$CLUSTER.yaml"
    sed -i "s|https://127\.0\.0\.1:|https://$REMOTE_HOST:|g" "$TMPDIR/$CLUSTER.yaml"
    sed -i "s|https://localhost:|https://$REMOTE_HOST:|g" "$TMPDIR/$CLUSTER.yaml"

    echo "  Server: $(grep server "$TMPDIR/$CLUSTER.yaml" | head -1 | xargs)"
done

echo ""
echo "Merging into ~/.kube/config..."

# Backup existing config
if [ -f "$HOME/.kube/config" ]; then
    cp "$HOME/.kube/config" "$HOME/.kube/config.backup.$(date +%Y%m%d%H%M%S)"
fi

# Merge all configs
KUBECONFIG="$HOME/.kube/config:$TMPDIR/babymilk-dev.yaml:$TMPDIR/babymilk-prod.yaml" \
    kubectl config view --flatten > "$TMPDIR/merged.yaml"

mv "$TMPDIR/merged.yaml" "$HOME/.kube/config"
chmod 600 "$HOME/.kube/config"

echo ""
echo "=== Kubeconfig merged ==="
echo ""
echo "Available contexts:"
kubectl config get-contexts -o name | grep -E "k3d-(babymilk|local)" || true
echo ""
echo "Switch context:"
echo "  kubectl config use-context k3d-babymilk-dev"
echo "  kubectl config use-context k3d-babymilk-prod"
echo "  kubectl config use-context k3d-local"
