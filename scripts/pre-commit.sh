#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

echo "Running pre-commit checks..."
echo ""

FAIL=0
STASHED_UNSTAGED=0

cleanup() {
    STATUS=$?
    trap - EXIT

    if [ "$STASHED_UNSTAGED" -eq 1 ]; then
        if ! git stash pop --quiet >/dev/null 2>&1; then
            echo "WARNING: could not automatically restore stashed unstaged changes."
            echo "Run 'git stash list' to recover them if needed."
        fi
    fi

    exit "$STATUS"
}

trap cleanup EXIT

echo "--- Secret Scan (staged changes) ---"
if ! "$SCRIPT_DIR/secret-scan.sh" staged; then
    FAIL=1
fi

HAS_UNSTAGED=0
if ! git diff --quiet --ignore-submodules -- || [ -n "$(git ls-files --others --exclude-standard)" ]; then
    HAS_UNSTAGED=1
fi

if [ "$HAS_UNSTAGED" -eq 1 ]; then
    echo ""
    echo "--- Stashing unstaged changes for staged-only checks ---"
    git stash push --keep-index --include-untracked --quiet -m "pre-commit staged-only checks"
    STASHED_UNSTAGED=1
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

echo ""
echo "--- Full CI Pipeline ---"
export MUTATION_DIFF_BASE=HEAD
if ! "$SCRIPT_DIR/ci.sh"; then
    FAIL=1
fi

if [ "$FAIL" -eq 0 ]; then
    echo "Pre-commit checks: PASSED"
else
    echo "Pre-commit checks: FAILED"
    exit 1
fi
