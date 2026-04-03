#!/usr/bin/env bash
set -euo pipefail

echo "========================================"
echo " Stage 4: Integration Tests"
echo "========================================"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

echo ""
echo "Running integration tests..."
go test ./... -tags=integration -count=1 -v

echo ""
echo "Integration test stage: PASSED"
