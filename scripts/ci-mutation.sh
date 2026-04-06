#!/usr/bin/env bash
set -euo pipefail

echo "========================================"
echo " Stage 3: Mutation Tests"
echo "========================================"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

MIN_MUTATION_SCORE=90
MUTATION_DIFF_BASE="${MUTATION_DIFF_BASE:-}"

detect_cpu_count() {
    local cpu_count=1

    if command -v getconf &>/dev/null; then
        cpu_count="$(getconf _NPROCESSORS_ONLN 2>/dev/null || printf '1')"
    elif command -v nproc &>/dev/null; then
        cpu_count="$(nproc 2>/dev/null || printf '1')"
    fi

    case "$cpu_count" in
        ''|*[!0-9]*) cpu_count=1 ;;
    esac

    printf '%s' "$cpu_count"
}

is_positive_int() {
    case "$1" in
        ''|*[!0-9]*) return 1 ;;
    esac

    [ "$1" -ge 1 ]
}

DEFAULT_MUTATION_WORKERS="$(detect_cpu_count)"
MUTATION_WORKERS="${MUTATION_WORKERS:-$DEFAULT_MUTATION_WORKERS}"
MUTATION_TEST_CPU="${MUTATION_TEST_CPU:-1}"

if ! is_positive_int "$MUTATION_WORKERS"; then
    MUTATION_WORKERS="$DEFAULT_MUTATION_WORKERS"
fi

if ! is_positive_int "$MUTATION_TEST_CPU"; then
    MUTATION_TEST_CPU=1
fi

GREMLINS_ARGS=()
if [ -n "$MUTATION_DIFF_BASE" ]; then
    echo "Using diff mode against: $MUTATION_DIFF_BASE"
    GREMLINS_ARGS=(--diff "$MUTATION_DIFF_BASE")
fi

GREMLINS_ARGS+=(--workers "$MUTATION_WORKERS" --test-cpu "$MUTATION_TEST_CPU")

echo ""
echo "Running mutation tests..."

if command -v gremlins &>/dev/null; then
    gremlins unleash "${GREMLINS_ARGS[@]}" --timeout-coefficient=3 --threshold-efficacy "$MIN_MUTATION_SCORE"

    echo ""
    echo "Mutation test stage: PASSED"
else
    echo "WARNING: gremlins not installed, skipping mutation tests"
    echo "Install from: https://github.com/go-gremlins/gremlins"
    echo "Mutation test stage: SKIPPED"
fi
