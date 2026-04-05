#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-repo}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

SECRET_REGEX='(-----BEGIN (RSA|EC|OPENSSH|PGP) PRIVATE KEY-----|AKIA[0-9A-Z]{16}|ASIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{36,255}|xox[baprs]-[A-Za-z0-9-]{10,200}|AIza[0-9A-Za-z_-]{35}|sk_(live|test)_[A-Za-z0-9]{24,}|(api[_-]?key|secret|token|password|passwd|private[_-]?key)[[:space:][:punct:]]+[A-Za-z0-9+/=_-]{20,})'

# Explicit synthetic fixtures used in tests are allowed so the scan can stay
# closed on real secrets without flagging known dummy values.
ALLOWED_MATCH_SUBSTRINGS=(
    "this-is-a-valid-admin-token-that-is-at-least-32-chars"
    "another-valid-token-that-is-at-least-32-characters-long"
)

if [ "$MODE" != "repo" ] && [ "$MODE" != "staged" ]; then
    echo "FAIL: unknown scan mode '$MODE'"
    exit 1
fi

GREP_ARGS=()
if [ "$MODE" = "staged" ]; then
    GREP_ARGS=(--cached)
fi

if MATCHES=$(git grep "${GREP_ARGS[@]}" -nI -E "$SECRET_REGEX" -- .); then
    FILTERED_MATCHES=""
    if [ -n "$MATCHES" ]; then
        while IFS= read -r line; do
            [ -z "$line" ] && continue

            allowed=0
            for allowed_match in "${ALLOWED_MATCH_SUBSTRINGS[@]}"; do
                case "$line" in
                    *"$allowed_match"*)
                        allowed=1
                        break
                        ;;
                esac
            done

            if [ "$allowed" -eq 0 ]; then
                FILTERED_MATCHES+="$line"$'\n'
            fi
        done <<< "$MATCHES"
    fi

    if [ -n "$FILTERED_MATCHES" ]; then
        echo "FAIL: potential secret material detected"
        printf '%s' "$FILTERED_MATCHES"
        exit 1
    fi
else
    STATUS=$?
    if [ "$STATUS" -gt 1 ]; then
        echo "FAIL: secret scan could not be completed"
        exit "$STATUS"
    fi
fi

echo "Secret scan: PASSED ($MODE)"
