#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

echo "Running pre-commit checks..."
echo ""

FAIL=0

echo "--- Secret Scan (staged changes) ---"
if ! "$SCRIPT_DIR/secret-scan.sh" staged; then
    FAIL=1
fi

echo "--- golangci-lint (with auto-fix) ---"
if command -v golangci-lint &>/dev/null; then
    if ! golangci-lint run --fix ./...; then
        echo "FAIL: golangci-lint found issues. Fix them and re-stage."
        FAIL=1
    fi
else
    echo "WARNING: golangci-lint not installed, skipping"
fi

echo ""
echo "--- File Size Budget ---"
MAX_LINES=500
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' | grep -v '_test\.go$' || true)

if [ -n "$STAGED_GO_FILES" ]; then
    while IFS= read -r file; do
        if [ -f "$file" ]; then
            LINES=$(wc -l < "$file")
            if [ "$LINES" -gt "$MAX_LINES" ]; then
                echo "FAIL: $file has $LINES lines (max: $MAX_LINES)"
                FAIL=1
            fi
        fi
    done <<< "$STAGED_GO_FILES"
fi

if [ "$FAIL" -eq 0 ]; then
    echo "Pre-commit checks: PASSED"
else
    echo "Pre-commit checks: FAILED"
    exit 1
fi
