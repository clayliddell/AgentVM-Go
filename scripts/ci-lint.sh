#!/usr/bin/env bash
set -euo pipefail

echo "========================================"
echo " Stage 1: Lint"
echo "========================================"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

echo ""
echo "--- golangci-lint ---"
if command -v golangci-lint &>/dev/null; then
    golangci-lint run ./...
    echo "golangci-lint: PASSED"
else
    echo "WARNING: golangci-lint not installed, skipping"
fi

echo ""
echo "--- Custom Analyzers ---"
ANALYZER_BIN="$ROOT_DIR/tools/analyzers/bin/analyzers"
if [ -f "$ANALYZER_BIN" ]; then
    echo "Building custom analyzers..."
    (cd "$ROOT_DIR/tools/analyzers" && go build -o "$ANALYZER_BIN" .)
fi

if [ -f "$ANALYZER_BIN" ]; then
    echo "Running custom analyzers..."
    "$ANALYZER_BIN" $(go list ./... | grep -v 'tools/analyzers')
    echo "Custom analyzers: PASSED"
else
    echo "WARNING: Custom analyzers binary not found at $ANALYZER_BIN"
    echo "Building from source..."
    (cd "$ROOT_DIR/tools/analyzers" && go build -o "$ANALYZER_BIN" .)
    "$ANALYZER_BIN" $(go list ./... | grep -v 'tools/analyzers')
    echo "Custom analyzers: PASSED"
fi

echo ""
echo "Lint stage: PASSED"
