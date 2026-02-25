#!/bin/bash
# MilkApp Test Runner
# Usage: ./scripts/test.sh [unit|integration|all|coverage|report]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

run_unit_tests() {
    echo -e "${GREEN}=== Running Unit Tests ===${NC}"
    go test ./... -v -count=1 2>&1 | tee test-output.txt
    echo -e "${GREEN}=== Unit Tests Complete ===${NC}"
}

run_unit_tests_with_coverage() {
    echo -e "${GREEN}=== Running Unit Tests with Coverage ===${NC}"
    go test ./... -v -count=1 -coverprofile=coverage.out -covermode=atomic 2>&1 | tee test-output.txt
    echo ""
    echo -e "${YELLOW}--- Coverage Summary ---${NC}"
    go tool cover -func=coverage.out
    echo ""
    echo -e "${YELLOW}Generating HTML coverage report: coverage.html${NC}"
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}=== Coverage Report Generated ===${NC}"
}

start_test_db() {
    echo -e "${YELLOW}Starting test database...${NC}"
    docker compose -f docker-compose.test.yml up -d --wait
    echo -e "${GREEN}Test database is ready${NC}"
}

stop_test_db() {
    echo -e "${YELLOW}Stopping test database...${NC}"
    docker compose -f docker-compose.test.yml down -v
    echo -e "${GREEN}Test database stopped${NC}"
}

run_integration_tests() {
    echo -e "${GREEN}=== Running Integration Tests ===${NC}"
    start_test_db
    export TEST_DATABASE_URL="postgres://testuser:testpass@localhost:5433/milkapp_test?sslmode=disable"
    go test ./... -v -count=1 -tags=integration -run "Integration|API_" 2>&1 | tee test-output-integration.txt
    local exit_code=${PIPESTATUS[0]}
    stop_test_db
    echo -e "${GREEN}=== Integration Tests Complete ===${NC}"
    return $exit_code
}

run_all_tests() {
    echo -e "${GREEN}=== Running All Tests ===${NC}"
    
    # Unit tests with coverage
    run_unit_tests_with_coverage
    
    # Integration tests
    start_test_db
    export TEST_DATABASE_URL="postgres://testuser:testpass@localhost:5433/milkapp_test?sslmode=disable"
    go test ./... -v -count=1 -tags=integration -coverprofile=coverage-integration.out -covermode=atomic 2>&1 | tee test-output-integration.txt
    stop_test_db
    
    echo -e "${GREEN}=== All Tests Complete ===${NC}"
}

generate_junit_report() {
    echo -e "${YELLOW}Generating JUnit XML report...${NC}"
    if ! command -v go-junit-report &> /dev/null; then
        echo "Installing go-junit-report..."
        go install github.com/jstemmer/go-junit-report/v2@latest
    fi
    
    if [ -f test-output.txt ]; then
        cat test-output.txt | go-junit-report -set-exit-code > test-results.xml
        echo -e "${GREEN}JUnit report generated: test-results.xml${NC}"
    else
        echo -e "${RED}No test output found. Run tests first.${NC}"
        exit 1
    fi
}

case "${1:-unit}" in
    unit)
        run_unit_tests
        ;;
    coverage)
        run_unit_tests_with_coverage
        ;;
    integration)
        run_integration_tests
        ;;
    all)
        run_all_tests
        ;;
    report)
        generate_junit_report
        ;;
    *)
        echo "Usage: $0 {unit|coverage|integration|all|report}"
        echo ""
        echo "  unit         Run unit tests only (default)"
        echo "  coverage     Run unit tests with coverage report"
        echo "  integration  Start test DB, run integration tests, stop test DB"
        echo "  all          Run unit tests with coverage + integration tests"
        echo "  report       Generate JUnit XML report from last test run"
        exit 1
        ;;
esac
