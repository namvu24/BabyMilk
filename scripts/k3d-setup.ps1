# Create k3d cluster with local Docker registry for MilkApp
$ErrorActionPreference = "Stop"

$RegistryName = "milkapp-registry"
$RegistryPort = 5111
$ClusterName = "milkapp"

# Create registry if it doesn't exist
$registries = k3d registry list -o json 2>$null | ConvertFrom-Json
if ($registries | Where-Object { $_.name -eq "k3d-$RegistryName" }) {
    Write-Host "Registry '$RegistryName' already exists, skipping..." -ForegroundColor Yellow
} else {
    Write-Host "Creating local Docker registry..." -ForegroundColor Green
    k3d registry create $RegistryName --port $RegistryPort
}

# Create cluster if it doesn't exist
$clusters = k3d cluster list -o json 2>$null | ConvertFrom-Json
if ($clusters | Where-Object { $_.name -eq $ClusterName }) {
    Write-Host "Cluster '$ClusterName' already exists, skipping..." -ForegroundColor Yellow
} else {
    Write-Host "Creating k3d cluster..." -ForegroundColor Green
    k3d cluster create $ClusterName `
        --registry-use "k3d-${RegistryName}:${RegistryPort}" `
        -p "8080:80@loadbalancer" `
        --agents 1
}

Write-Host ""
Write-Host "=== k3d environment ready ===" -ForegroundColor Green
Write-Host "Cluster:  $ClusterName"
Write-Host "Registry: localhost:$RegistryPort (cluster: k3d-${RegistryName}:${RegistryPort})"
Write-Host "Ingress:  http://localhost:8080"
Write-Host ""
Write-Host "Next: run build-image script, then deploy-local script"
