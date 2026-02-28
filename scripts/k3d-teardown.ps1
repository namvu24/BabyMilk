# Tear down k3d cluster for BabyMilk
$ErrorActionPreference = "Stop"

$ClusterName = "babymilk"
$Namespace = "babymilk"

Write-Host "=== Tearing down BabyMilk k3d environment ===" -ForegroundColor Yellow

# Uninstall Helm release and delete cluster
$clusters = k3d cluster list -o json 2>$null | ConvertFrom-Json
if ($clusters | Where-Object { $_.name -eq $ClusterName }) {
    Write-Host "Uninstalling Helm release..." -ForegroundColor Yellow
    helm uninstall babymilk -n $Namespace 2>$null
    
    Write-Host "Deleting k3d cluster..." -ForegroundColor Yellow
    k3d cluster delete $ClusterName
} else {
    Write-Host "Cluster '$ClusterName' not found, skipping..." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Teardown complete ===" -ForegroundColor Green
