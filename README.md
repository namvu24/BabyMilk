# BabyMilk - Baby Milk Consumption Tracker

A simple web app to track your baby's milk consumption, built with Go, PostgreSQL, and vanilla HTML/JS.

## Project Structure

```
babymilk/
├── cmd/
│   └── server/
│       ├── main.go              # Entry point, HTTP server & routing
│       └── main_test.go         # CORS middleware tests
├── internal/
│   └── app/
│       ├── api_integration_test.go  # API integration tests
│       ├── db.go                # Database connection and repository
│       ├── db_integration_test.go   # DB repository integration tests
│       ├── handlers.go          # REST API handlers
│       ├── handlers_test.go     # Handler unit tests (mocked repository)
│       ├── migrate.go           # Auto-create tables on startup
│       ├── models.go            # Data models and validation
│       ├── models_test.go       # Validation unit tests
│       └── repository.go       # Repository interface
├── scripts/
│   ├── test.sh                  # Test runner (Linux/macOS)
│   └── test.ps1                 # Test runner (Windows)
├── static/
│   ├── index.html               # Main page (form + table + chart)
│   ├── app.js                   # Frontend logic
│   └── style.css                # Custom styles
├── deploy/
│   └── k8s/                     # Kubernetes manifests
├── docker-compose.yml
├── docker-compose.test.yml      # Test database (PostgreSQL)
├── Dockerfile
├── .dockerignore
├── go.mod
└── README.md
```

## Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [PostgreSQL](https://www.postgresql.org/download/)
- [Docker](https://docs.docker.com/get-docker/) (optional, for containerized deployment)

## Setup

### 1. Create the PostgreSQL database

```sql
CREATE DATABASE babymilk;
```

### 2. Set environment variables (optional)

| Variable       | Default                                                        | Description             |
|----------------|----------------------------------------------------------------|-------------------------|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/babymilk?sslmode=disable` | PostgreSQL connection string |
| `PORT`         | `8080`                                                         | HTTP server port        |

### 3. Build and run

```bash
go build -o babymilk.exe ./cmd/server
.\babymilk.exe
```

Or run directly:

```bash
go run ./cmd/server
```

The server will start at **http://localhost:8080**.

### 4. Quick start with Docker Compose

```bash
docker compose up --build
```

This starts PostgreSQL and the app together. Open **http://localhost:8080**.

To stop and remove containers:

```bash
docker compose down
```

To also delete the database volume:

```bash
docker compose down -v
```

### 5. Local Kubernetes with k3d

For a production-like local setup using Kubernetes:

```bash
# Prerequisites: Docker (logged in to Docker Hub), k3d, helm, kubectl

# 1. Create k3d cluster
.\scripts\k3d-setup.ps1          # Windows
# ./scripts/k3d-setup.sh         # Linux/macOS

# 2. Build, push, and deploy (all-in-one)
.\scripts\deploy-local.ps1       # Windows
# ./scripts/deploy-local.sh      # Linux/macOS

# 3. Open http://localhost:8080

# Teardown
.\scripts\k3d-teardown.ps1       # Windows
# ./scripts/k3d-teardown.sh      # Linux/macOS
```

The Helm chart and K8s manifests are in the separate [`babymilk-deploy`](../babymilk-deploy/) repo for GitOps workflows. See its [README](../babymilk-deploy/README.md) for full details.

### Multi-Environment Setup (homelab)

Deploy dev and prod environments on a remote laptop (`homelab`) with ArgoCD:

```bash
# 1. Create k3d clusters on homelab (via SSH)
./scripts/k3d-remote-setup.sh

# 2. Merge kubeconfigs into main machine
./scripts/kubeconfig-setup.sh          # Linux/macOS
# .\scripts\kubeconfig-setup.ps1       # Windows

# 3. Install ArgoCD and apply Application manifests
./scripts/argocd-setup.sh

# Switch between environments
kubectl config use-context k3d-babymilk-dev
kubectl config use-context k3d-babymilk-prod
kubectl config use-context k3d-babymilk          # local testing

# Access apps
# Local:  http://localhost:8080
# Dev:    http://homelab:8081
# Prod:   http://homelab:8082

# ArgoCD UI (port-forward)
kubectl --context k3d-babymilk-dev port-forward svc/argocd-server -n argocd 9080:443
# Then open: https://localhost:9080

# Teardown remote clusters
./scripts/k3d-remote-teardown.sh
```

## Features

- **Add feedings** — record amount (ml), start time, and end time
- **Edit/Delete** — correct mistakes in existing entries
- **Daily chart** — bar chart showing total ml consumed per day (last 7 days)
- **Date filter** — filter the feedings table by date

## API Endpoints

| Method | Path                 | Description                        |
|--------|----------------------|------------------------------------|
| GET    | `/api/feedings`      | List feedings (optional `?date=YYYY-MM-DD`) |
| POST   | `/api/feedings`      | Create a new feeding               |
| PUT    | `/api/feedings/{id}` | Update a feeding                   |
| DELETE | `/api/feedings/{id}` | Delete a feeding                   |
| GET    | `/api/feedings/daily`| Daily totals (optional `?days=N`)  |

## Testing

MilkApp uses Go's built-in testing framework. Tests are split into **unit tests** (no external dependencies) and **integration tests** (require PostgreSQL via Docker).

### Run unit tests

```bash
go test ./... -v
```

Or use the test script:

```bash
# Linux/macOS
./scripts/test.sh unit

# Windows (PowerShell)
.\scripts\test.ps1 -Mode unit
```

### Run unit tests with coverage

```bash
go test ./... -v -coverprofile=coverage.out -covermode=atomic

# View coverage summary in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

Or use the test script:

```bash
# Linux/macOS
./scripts/test.sh coverage

# Windows (PowerShell)
.\scripts\test.ps1 -Mode coverage
```

Open `coverage.html` in a browser to view a visual coverage report.

### Run integration tests

## CI/CD

Automated CI/CD is handled by GitHub Actions. Three workflows exist:

| Workflow | Trigger | Actions |
|----------|---------|---------|
| **CI** (`ci.yml`) | PR to `main` | Run unit + integration tests |
| **Build & Push** (`build-push.yml`) | Push to `main` | Test → build → push image (`main-<sha>` + `latest`) → update dev |
| **Release** (`release.yml`) | Tag `v*` | Test → build → push image (`v1.x.x` + `stable`) → update prod → GitHub Release |

### Required GitHub Secrets

Configure these in the BabyMilk repo **Settings → Secrets and variables → Actions**:

| Secret | Description |
|--------|-------------|
| `DOCKERHUB_USERNAME` | Docker Hub username (e.g., `namvu24`) |
| `DOCKERHUB_TOKEN` | Docker Hub access token ([create here](https://hub.docker.com/settings/security)) |
| `DEPLOY_REPO_TOKEN` | GitHub PAT with `repo` scope and write access to `namvu24/babymilk-deploy` |

### Creating a release

```bash
git tag v1.0.0
git push origin v1.0.0
```

This triggers the Release workflow which builds the image, pushes it as `v1.0.0` + `stable`, updates the production values in `babymilk-deploy`, and creates a GitHub Release with auto-generated notes.

Integration tests require Docker to spin up a PostgreSQL container.

```bash
# Start the test database
docker compose -f docker-compose.test.yml up -d --wait

# Run integration tests
DATABASE_URL="postgres://testuser:testpass@localhost:5433/babymilk_test?sslmode=disable" \
  go test ./... -v -count=1 -tags=integration

# Stop and clean up the test database
docker compose -f docker-compose.test.yml down -v
```

Or use the test script (handles start/stop automatically):

```bash
# Linux/macOS
./scripts/test.sh integration

# Windows (PowerShell)
.\scripts\test.ps1 -Mode integration
```

### Run all tests (unit + integration)

```bash
# Linux/macOS
./scripts/test.sh all

# Windows (PowerShell)
.\scripts\test.ps1 -Mode all
```

### Generate JUnit XML report

JUnit XML reports are useful for CI/CD pipelines (e.g., GitHub Actions, Azure DevOps).

First, install [`go-junit-report`](https://github.com/jstemmer/go-junit-report):

```bash
go install github.com/jstemmer/go-junit-report/v2@latest
```

Then generate the report:

```bash
go test ./... -v 2>&1 | go-junit-report -set-exit-code > test-results.xml
```

Or use the test script (run tests first, then generate the report):

```bash
# Linux/macOS
./scripts/test.sh unit      # run tests first
./scripts/test.sh report    # generate JUnit XML from last run

# Windows (PowerShell)
.\scripts\test.ps1 -Mode unit
.\scripts\test.ps1 -Mode report
```

### Test architecture

| Type | Build tag | Files | Database required |
|------|-----------|-------|-------------------|
| Unit tests | *(none)* | `*_test.go` (without `//go:build integration`) | No |
| Integration tests | `integration` | `*_integration_test.go` | Yes (PostgreSQL via Docker) |

- **Unit tests** use a mock repository to test handlers and validation in isolation.
- **Integration tests** use `//go:build integration` so they are skipped by default. Pass `-tags=integration` to include them.
- The test database runs on **port 5433** to avoid conflicts with a development PostgreSQL on port 5432.
