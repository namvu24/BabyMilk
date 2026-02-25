# Tear down k3d cluster and registry for MilkApp
$ErrorActionPreference = "Stop"

$ClusterName = "milkapp"
$RegistryName = "milkapp-registry"
$Namespace = "milkapp"

Write-Host "=== Tearing down MilkApp k3d environment ===" -ForegroundColor Yellow

# Uninstall Helm release (if cluster is running)
$clusters = k3d cluster list -o json 2>$null | ConvertFrom-Json
if ($clusters | Where-Object { $_.name -eq $ClusterName }) {
    Write-Host "Uninstalling Helm release..." -ForegroundColor Yellow
    helm uninstall milkapp -n $Namespace 2>$null
    
    Write-Host "Deleting k3d cluster..." -ForegroundColor Yellow
    k3d cluster delete $ClusterName
} else {
    Write-Host "Cluster '$ClusterName' not found, skipping..." -ForegroundColor Yellow
}

# Delete registry
$registries = k3d registry list -o json 2>$null | ConvertFrom-Json
if ($registries | Where-Object { $_.name -eq "k3d-$RegistryName" }) {
    Write-Host "Deleting Docker registry..." -ForegroundColor Yellow
    k3d registry delete "k3d-$RegistryName"
} else {
    Write-Host "Registry '$RegistryName' not found, skipping..." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Teardown complete ===" -ForegroundColor Green
