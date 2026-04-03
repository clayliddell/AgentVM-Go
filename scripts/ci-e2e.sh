#!/usr/bin/env bash
set -euo pipefail

echo "========================================"
echo " Stage 7: E2E Tests"
echo "========================================"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

if [ "${CI_E2E:-}" != "true" ]; then
    echo "E2E tests skipped (set CI_E2E=true to run)"
    echo "E2E test stage: SKIPPED"
    exit 0
fi

echo ""
echo "Running E2E tests..."
go test ./... -tags=e2e -count=1 -v

echo ""
echo "E2E test stage: PASSED"
