# Deploy MilkApp to local k3d cluster
param(
    [string]$Tag = "latest"
)

$ErrorActionPreference = "Stop"

$ScriptDir = $PSScriptRoot
$ProjectDir = Split-Path $ScriptDir -Parent
$DeployDir = Join-Path (Split-Path $ProjectDir -Parent) "milkapp-deploy"
$ChartDir = Join-Path $DeployDir "charts\milkapp"
$Namespace = "milkapp"
$ReleaseName = "milkapp"

Write-Host "=== MilkApp Local Deployment ===" -ForegroundColor Green
Write-Host ""

# Step 1: Build and push image
Write-Host "[1/3] Building and pushing Docker image..." -ForegroundColor Yellow
& "$ScriptDir\build-image.ps1" -Tag $Tag
Write-Host ""

# Step 2: Deploy with Helm
Write-Host "[2/3] Deploying with Helm..." -ForegroundColor Yellow
helm upgrade --install $ReleaseName $ChartDir `
    -f "$ChartDir\values-local.yaml" `
    -f "$ChartDir\values-local-secrets.yaml" `
    --set "image.tag=$Tag" `
    -n $Namespace --create-namespace `
    --wait --timeout 5m
Write-Host ""

# Step 3: Wait and show status
Write-Host "[3/3] Verifying deployment..." -ForegroundColor Yellow
kubectl get pods -n $Namespace
Write-Host ""
Write-Host "=== Deployment complete ===" -ForegroundColor Green
Write-Host "Access MilkApp at: http://localhost:8080"
Write-Host ""
Write-Host "Useful commands:"
Write-Host "  kubectl get all -n $Namespace"
Write-Host "  kubectl logs -n $Namespace -l app.kubernetes.io/name=milkapp"
Write-Host "  helm status $ReleaseName -n $Namespace"
