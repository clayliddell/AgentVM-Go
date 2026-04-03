#!/usr/bin/env bash
set -euo pipefail

echo "========================================"
echo " Stage 6: Budget Checks"
echo "========================================"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

MAX_LINES=500
MAX_FILES=10
FAIL=0

echo ""
echo "--- File Size Budget (max ${MAX_LINES} lines per .go file, excluding tests) ---"
while IFS= read -r -d '' file; do
    LINES=$(wc -l < "$file")
    if [ "$LINES" -gt "$MAX_LINES" ]; then
        echo "FAIL: $file has $LINES lines (max: $MAX_LINES)"
        FAIL=1
    fi
done < <(find "$ROOT_DIR" -name "*.go" ! -name "*_test.go" -print0)

if [ "$FAIL" -eq 0 ]; then
    echo "File size budget: PASSED"
fi

echo ""
echo "--- File Count Budget (max ${MAX_FILES} non-test .go files per package) ---"
while IFS= read -r -d '' dir; do
    COUNT=$(find "$dir" -maxdepth 1 -name "*.go" ! -name "*_test.go" | wc -l)
    if [ "$COUNT" -gt "$MAX_FILES" ]; then
        echo "FAIL: $dir has $COUNT non-test .go files (max: $MAX_FILES)"
        FAIL=1
    fi
done < <(find "$ROOT_DIR" -type d -name "*.go" -exec dirname {} \; 2>/dev/null | sort -u | tr '\n' '\0')

if [ "$FAIL" -eq 0 ]; then
    echo "File count budget: PASSED"
fi

echo ""
if [ "$FAIL" -ne 0 ]; then
    echo "Budget stage: FAILED"
    exit 1
fi

echo "Budget stage: PASSED"
