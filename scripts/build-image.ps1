# Build and push MilkApp Docker image to Docker Hub
param(
    [string]$Tag = "latest"
)

$ErrorActionPreference = "Stop"

$Image = "namvu24/babymilk:$Tag"
$ProjectDir = Split-Path $PSScriptRoot -Parent

# Check Docker Hub login
$dockerInfo = docker info 2>$null
if ($dockerInfo -notmatch "Username") {
    Write-Host "Please log in to Docker Hub first:" -ForegroundColor Yellow
    docker login
}

Write-Host "Building Docker image: $Image" -ForegroundColor Green
docker build -t $Image $ProjectDir

Write-Host "Pushing to Docker Hub..." -ForegroundColor Green
docker push $Image

Write-Host ""
Write-Host "=== Image pushed successfully ===" -ForegroundColor Green
Write-Host "Image: $Image"
