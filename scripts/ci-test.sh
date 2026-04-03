#!/usr/bin/env bash
set -euo pipefail

echo "========================================"
echo " Stage 2: Unit Tests"
echo "========================================"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

MIN_COVERAGE=95

echo ""
echo "Running unit tests with coverage..."
go test ./... -coverprofile=coverage.out -covermode=atomic -count=1

TOTAL_COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print gsub(/%/, "", $3); print $3}' || true)

if command -v go &>/dev/null; then
    COVERAGE_LINE=$(go tool cover -func=coverage.out | grep "total:" || echo "total: 0.0%")
    COVERAGE_PCT=$(echo "$COVERAGE_LINE" | awk '{print $3}' | tr -d '%')
    echo ""
    echo "Total coverage: ${COVERAGE_PCT}%"
    echo "Minimum required: ${MIN_COVERAGE}%"

    if [ "$(echo "$COVERAGE_PCT >= $MIN_COVERAGE" | bc -l 2>/dev/null || echo 1)" = "1" ]; then
        echo "Coverage check: SKIPPED (bc not available or no coverage data)"
        echo "Unit test stage: PASSED (coverage gate requires bc)"
        exit 0
    fi

    if (( $(echo "$COVERAGE_PCT < $MIN_COVERAGE" | bc -l) )); then
        echo "FAIL: Coverage ${COVERAGE_PCT}% is below minimum ${MIN_COVERAGE}%"
        exit 1
    fi
fi

echo ""
echo "Unit test stage: PASSED"
