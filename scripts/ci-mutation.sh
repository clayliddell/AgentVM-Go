#!/usr/bin/env bash
set -euo pipefail

echo "========================================"
echo " Stage 3: Mutation Tests"
echo "========================================"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

MIN_MUTATION_SCORE=90

echo ""
echo "Running mutation tests..."

if command -v go-mutesting &>/dev/null; then
    MUTATION_OUTPUT=$(go-mutesting ./... 2>&1 || true)
    echo "$MUTATION_OUTPUT"

    KILLED=$(echo "$MUTATION_OUTPUT" | grep -oP 'Killed: \K\d+' || echo "0")
    SURVIVED=$(echo "$MUTATION_OUTPUT" | grep -oP 'Survived: \K\d+' || echo "0")
    TOTAL=$((KILLED + SURVIVED))

    if [ "$TOTAL" -gt 0 ]; then
        SCORE=$(echo "scale=2; $KILLED * 100 / $TOTAL" | bc)
        echo ""
        echo "Mutation score: ${SCORE}%"
        echo "Minimum required: ${MIN_MUTATION_SCORE}%"

        if (( $(echo "$SCORE < $MIN_MUTATION_SCORE" | bc -l) )); then
            echo "FAIL: Mutation score ${SCORE}% is below minimum ${MIN_MUTATION_SCORE}%"
            exit 1
        fi
    else
        echo "No mutations found to test"
    fi

    echo ""
    echo "Mutation test stage: PASSED"
else
    echo "WARNING: go-mutesting not installed, skipping mutation tests"
    echo "Install with: go install github.com/avito-tech/go-mutesting/cmd/go-mutesting@latest"
    echo "Mutation test stage: SKIPPED"
fi
