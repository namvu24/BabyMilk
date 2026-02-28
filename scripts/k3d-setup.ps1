# Create k3d cluster for MilkApp
$ErrorActionPreference = "Stop"

$ClusterName = "milkapp"

# Create cluster if it doesn't exist
$clusters = k3d cluster list -o json 2>$null | ConvertFrom-Json
if ($clusters | Where-Object { $_.name -eq $ClusterName }) {
    Write-Host "Cluster '$ClusterName' already exists, skipping..." -ForegroundColor Yellow
} else {
    Write-Host "Creating k3d cluster..." -ForegroundColor Green
    k3d cluster create $ClusterName `
        -p "8080:80@loadbalancer" `
        --agents 1
}

Write-Host ""
Write-Host "=== k3d environment ready ===" -ForegroundColor Green
Write-Host "Cluster:  $ClusterName"
Write-Host "Ingress:  http://localhost:8080"
Write-Host ""
Write-Host "Next: run build-image script, then deploy-local script"
