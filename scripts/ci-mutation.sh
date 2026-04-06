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

run_mutation_stage() {
    local stage_dir="$1"
    local stage_name="$2"
    shift 2

    local stage_extra_args=("$@")
    local gremlins_output=""
    local gremlins_status=0
    local mutator_coverage=""
    local test_efficacy=""

    echo ""
    echo "Running mutation tests (${stage_name})..."

    if command -v gremlins &>/dev/null; then
        set +e
        gremlins_output="$(cd "$stage_dir" && gremlins unleash . "${stage_extra_args[@]}" "${GREMLINS_ARGS[@]}" --timeout-coefficient=3 --threshold-mcover "$MIN_MUTATION_SCORE" --threshold-efficacy "$MIN_MUTATION_SCORE" 2>&1)"
        gremlins_status=$?
        set -e

        printf '%s\n' "$gremlins_output"

        mutator_coverage="$(printf '%s\n' "$gremlins_output" | awk -F': ' '/Mutator coverage:/ {gsub(/%/, "", $2); print $2; exit}')"
        test_efficacy="$(printf '%s\n' "$gremlins_output" | awk -F': ' '/Test efficacy:/ {gsub(/%/, "", $2); print $2; exit}')"

        if [ "$gremlins_status" -ne 0 ]; then
            echo ""
            echo "Mutation test stage (${stage_name}): FAILED"
            exit "$gremlins_status"
        fi

        if [ -z "$mutator_coverage" ] || [ -z "$test_efficacy" ]; then
            echo ""
            echo "FAIL: Unable to parse gremlins mutation metrics for ${stage_name}"
            echo "Mutation test stage (${stage_name}): FAILED"
            exit 1
        fi

        if ! awk -v actual="$mutator_coverage" -v minimum="$MIN_MUTATION_SCORE" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'; then
            echo ""
            echo "FAIL: Mutator coverage ${mutator_coverage}% is below minimum ${MIN_MUTATION_SCORE}% (${stage_name})"
            echo "Mutation test stage (${stage_name}): FAILED"
            exit 1
        fi

        if ! awk -v actual="$test_efficacy" -v minimum="$MIN_MUTATION_SCORE" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'; then
            echo ""
            echo "FAIL: Test efficacy ${test_efficacy}% is below minimum ${MIN_MUTATION_SCORE}% (${stage_name})"
            echo "Mutation test stage (${stage_name}): FAILED"
            exit 1
        fi

        echo ""
        echo "Mutation test stage (${stage_name}): PASSED"
    else
        echo "WARNING: gremlins not installed, skipping mutation tests (${stage_name})"
        echo "Install from: https://github.com/go-gremlins/gremlins"
        echo "Mutation test stage (${stage_name}): SKIPPED"
    fi
}

echo ""
run_mutation_stage "$ROOT_DIR" "root" --exclude-files '^tools/analyzers/'
run_mutation_stage "$ROOT_DIR/tools/analyzers" "analyzers" --coverpkg ./...
