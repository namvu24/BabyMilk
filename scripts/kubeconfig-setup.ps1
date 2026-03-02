# Fetch kubeconfigs from homelab k3d clusters and merge into local ~/.kube/config
# Run from main machine after k3d-remote-setup.sh
$ErrorActionPreference = "Stop"

$Remote = "nam@homelab"
$RemoteHost = "homelab"

Write-Host "=== Fetching kubeconfigs from $RemoteHost ===" -ForegroundColor Green
Write-Host ""

$TmpDir = Join-Path $env:TEMP "kubeconfig-$(Get-Random)"
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    foreach ($Cluster in @("babymilk-dev", "babymilk-prod")) {
        Write-Host "Fetching kubeconfig for $Cluster..." -ForegroundColor Yellow
        # Get kubeconfig from remote
        ssh $Remote "sudo k3d kubeconfig get $Cluster" | Out-File -FilePath "$TmpDir\$Cluster.yaml" -Encoding utf8

        # Replace localhost/0.0.0.0 with homelab hostname
        $content = Get-Content "$TmpDir\$Cluster.yaml" -Raw
        $content = $content -replace 'https://0\.0\.0\.0:', "https://${RemoteHost}:"
        $content = $content -replace 'https://127\.0\.0\.1:', "https://${RemoteHost}:"
        $content = $content -replace 'https://localhost:', "https://${RemoteHost}:"
        $content | Set-Content "$TmpDir\$Cluster.yaml" -Encoding utf8

        $server = (Select-String -Path "$TmpDir\$Cluster.yaml" -Pattern "server:" | Select-Object -First 1).Line.Trim()
        Write-Host "  $server"
    }

    Write-Host ""
    Write-Host "Merging into ~/.kube/config..." -ForegroundColor Yellow

    # Backup existing config
    $KubeDir = Join-Path $env:USERPROFILE ".kube"
    $KubeConfig = Join-Path $KubeDir "config"
    if (Test-Path $KubeConfig) {
        $backup = "$KubeConfig.backup.$(Get-Date -Format 'yyyyMMddHHmmss')"
        Copy-Item $KubeConfig $backup
    }

    # Merge all configs
    $env:KUBECONFIG = "$KubeConfig;$TmpDir\babymilk-dev.yaml;$TmpDir\babymilk-prod.yaml"
    kubectl config view --flatten | Out-File -FilePath "$TmpDir\merged.yaml" -Encoding utf8
    Copy-Item "$TmpDir\merged.yaml" $KubeConfig -Force
    $env:KUBECONFIG = $KubeConfig

    Write-Host ""
    Write-Host "=== Kubeconfig merged ===" -ForegroundColor Green
    Write-Host ""
    Write-Host "Available contexts:"
    kubectl config get-contexts -o name
    Write-Host ""
    Write-Host "Switch context:"
    Write-Host "  kubectl config use-context k3d-babymilk-dev"
    Write-Host "  kubectl config use-context k3d-babymilk-prod"
    Write-Host "  kubectl config use-context k3d-local"
}
finally {
    Remove-Item -Recurse -Force $TmpDir -ErrorAction SilentlyContinue
}
