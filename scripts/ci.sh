#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

START_TIME=$(date +%s)

echo "╔══════════════════════════════════════════════════════════╗"
echo "║                 CI Pipeline — AgentVM                    ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

STAGES=(
    "lint:CI Lint (golangci-lint + custom analyzers)"
    "test:Unit Tests (coverage gate >=95%)"
    "mutation:Mutation Tests (score gate >=90%)"
    "integration:Integration Tests"
    "security:Security Scanning (gosec + govulncheck)"
    "budgets:Budget Checks (file size + file count)"
    "e2e:E2E Tests (staging only)"
)

FAILED=()
PASSED=()

for stage in "${STAGES[@]}"; do
    NAME="${stage%%:*}"
    DESC="${stage##*:}"

    echo ""
    echo "▶ Running: $DESC"
    echo "──────────────────────────────────────────────────────"

    if "$SCRIPT_DIR/ci-${NAME}.sh"; then
        PASSED+=("$NAME")
        echo "✓ $NAME: PASSED"
    else
        FAILED+=("$NAME")
        echo "✗ $NAME: FAILED"
        echo ""
        echo "Pipeline failed at stage: $NAME"
        echo "Remaining stages skipped."
        break
    fi
done

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo "══════════════════════════════════════════════════════════"
echo " Results"
echo "══════════════════════════════════════════════════════════"

for p in "${PASSED[@]}"; do
    echo "  ✓ $p"
done

if [ ${#FAILED[@]} -gt 0 ]; then
    for f in "${FAILED[@]}"; do
        echo "  ✗ $f"
    done
    echo ""
    echo "Pipeline: FAILED (${#PASSED[@]}/${#STAGES[@]} stages passed)"
    echo "Duration: ${DURATION}s"
    exit 1
fi

echo ""
echo "Pipeline: ALL STAGES PASSED (${#PASSED[@]}/${#STAGES[@]})"
echo "Duration: ${DURATION}s"
