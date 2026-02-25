# Build and push MilkApp Docker image to local registry
param(
    [string]$Tag = "latest"
)

$ErrorActionPreference = "Stop"

$Registry = "localhost:5111"
$Image = "$Registry/milkapp:$Tag"
$ProjectDir = Split-Path $PSScriptRoot -Parent

Write-Host "Building Docker image: $Image" -ForegroundColor Green
docker build -t $Image $ProjectDir

Write-Host "Pushing to local registry..." -ForegroundColor Green
docker push $Image

Write-Host ""
Write-Host "=== Image pushed successfully ===" -ForegroundColor Green
Write-Host "Image: $Image"
Write-Host "Cluster ref: k3d-milkapp-registry:5111/milkapp:$Tag"
