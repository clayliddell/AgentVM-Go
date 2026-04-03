#!/usr/bin/env bash
set -euo pipefail

echo "========================================"
echo " Stage 5: Security Scanning"
echo "========================================"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

echo ""
echo "--- gosec ---"
if command -v gosec &>/dev/null; then
    gosec -exclude-generated ./...
    echo "gosec: PASSED"
else
    echo "WARNING: gosec not installed, skipping"
    echo "Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi

echo ""
echo "--- govulncheck ---"
if command -v govulncheck &>/dev/null; then
    govulncheck ./...
    echo "govulncheck: PASSED"
else
    echo "WARNING: govulncheck not installed, skipping"
    echo "Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

echo ""
echo "Security stage: PASSED"
