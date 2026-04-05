#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-repo}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

SECRET_REGEX='(-----BEGIN (RSA|EC|OPENSSH|PGP) PRIVATE KEY-----|AKIA[0-9A-Z]{16}|ASIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{36,255}|xox[baprs]-[A-Za-z0-9-]{10,200}|AIza[0-9A-Za-z_-]{35}|sk_(live|test)_[A-Za-z0-9]{24,}|(api[_-]?key|secret|token|password|passwd|private[_-]?key)[[:space:][:punct:]]+[A-Za-z0-9+/=_-]{20,})'

if [ "$MODE" != "repo" ] && [ "$MODE" != "staged" ]; then
    echo "FAIL: unknown scan mode '$MODE'"
    exit 1
fi

GREP_ARGS=()
if [ "$MODE" = "staged" ]; then
    GREP_ARGS=(--cached)
fi

if MATCHES=$(git grep "${GREP_ARGS[@]}" -nI -E "$SECRET_REGEX" -- . 2>/dev/null | grep -v '_test\.go:' || true); then
    if [ -n "$MATCHES" ]; then
        echo "FAIL: potential secret material detected"
        echo "$MATCHES"
        exit 1
    fi
fi

echo "Secret scan: PASSED ($MODE)"
