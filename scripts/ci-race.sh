#!/usr/bin/env bash
set -euo pipefail

echo "========================================"
echo " Stage 8: Race Condition Testing"
echo "========================================"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

echo ""
echo "Running tests with -race flag..."
if go test -race ./... -count=1; then
    echo ""
    echo "Race condition test stage: PASSED"
else
    echo ""
    echo "Race condition test stage: FAILED"
    exit 1
fi
