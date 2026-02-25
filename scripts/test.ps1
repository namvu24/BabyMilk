# MilkApp Test Runner (PowerShell)
# Usage: .\scripts\test.ps1 [-Mode unit|integration|all|coverage|report]

param(
    [ValidateSet("unit", "integration", "all", "coverage", "report")]
    [string]$Mode = "unit"
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path (Split-Path $PSScriptRoot -Parent) -Leaf -Resolve) -ErrorAction SilentlyContinue
Set-Location (Join-Path $PSScriptRoot "..")

function Run-UnitTests {
    Write-Host "=== Running Unit Tests ===" -ForegroundColor Green
    go test ./... -v -count=1 2>&1 | Tee-Object -FilePath test-output.txt
    Write-Host "=== Unit Tests Complete ===" -ForegroundColor Green
}

function Run-UnitTestsWithCoverage {
    Write-Host "=== Running Unit Tests with Coverage ===" -ForegroundColor Green
    go test ./... -v -count=1 -coverprofile=coverage.out -covermode=atomic 2>&1 | Tee-Object -FilePath test-output.txt
    Write-Host ""
    Write-Host "--- Coverage Summary ---" -ForegroundColor Yellow
    go tool cover -func=coverage.out
    Write-Host ""
    Write-Host "Generating HTML coverage report: coverage.html" -ForegroundColor Yellow
    go tool cover -html=coverage.out -o coverage.html
    Write-Host "=== Coverage Report Generated ===" -ForegroundColor Green
}

function Start-TestDB {
    Write-Host "Starting test database..." -ForegroundColor Yellow
    docker compose -f docker-compose.test.yml up -d --wait
    Write-Host "Test database is ready" -ForegroundColor Green
}

function Stop-TestDB {
    Write-Host "Stopping test database..." -ForegroundColor Yellow
    docker compose -f docker-compose.test.yml down -v
    Write-Host "Test database stopped" -ForegroundColor Green
}

function Run-IntegrationTests {
    Write-Host "=== Running Integration Tests ===" -ForegroundColor Green
    Start-TestDB
    $env:TEST_DATABASE_URL = "postgres://testuser:testpass@localhost:5433/milkapp_test?sslmode=disable"
    try {
        go test ./... -v -count=1 -tags=integration -run "Integration|API_" 2>&1 | Tee-Object -FilePath test-output-integration.txt
    }
    finally {
        Stop-TestDB
    }
    Write-Host "=== Integration Tests Complete ===" -ForegroundColor Green
}

function Run-AllTests {
    Write-Host "=== Running All Tests ===" -ForegroundColor Green

    Run-UnitTestsWithCoverage

    Start-TestDB
    $env:TEST_DATABASE_URL = "postgres://testuser:testpass@localhost:5433/milkapp_test?sslmode=disable"
    try {
        go test ./... -v -count=1 -tags=integration -coverprofile=coverage-integration.out -covermode=atomic 2>&1 | Tee-Object -FilePath test-output-integration.txt
    }
    finally {
        Stop-TestDB
    }

    Write-Host "=== All Tests Complete ===" -ForegroundColor Green
}

function Generate-JUnitReport {
    Write-Host "Generating JUnit XML report..." -ForegroundColor Yellow

    $junitReport = Get-Command go-junit-report -ErrorAction SilentlyContinue
    if (-not $junitReport) {
        Write-Host "Installing go-junit-report..." -ForegroundColor Yellow
        go install github.com/jstemmer/go-junit-report/v2@latest
    }

    if (Test-Path test-output.txt) {
        Get-Content test-output.txt | go-junit-report -set-exit-code > test-results.xml
        Write-Host "JUnit report generated: test-results.xml" -ForegroundColor Green
    }
    else {
        Write-Host "No test output found. Run tests first." -ForegroundColor Red
        exit 1
    }
}

switch ($Mode) {
    "unit"        { Run-UnitTests }
    "coverage"    { Run-UnitTestsWithCoverage }
    "integration" { Run-IntegrationTests }
    "all"         { Run-AllTests }
    "report"      { Generate-JUnitReport }
}
