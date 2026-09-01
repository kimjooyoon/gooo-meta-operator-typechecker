#!/usr/bin/env bash
set -euo pipefail

test "$(git status --porcelain)" = ""
if command -v rg >/dev/null 2>&1; then
  count_files() { rg -l "$1" "$2" --glob "$3" | wc -l | tr -d ' '; }
  count_lines() { rg "$1" "$2" --glob "$3" | wc -l | tr -d ' '; }
else
  count_files() { grep -RIl --include="*${3#\*}" "$1" "$2" | wc -l | tr -d ' '; }
  count_lines() { grep -RIl --include="*${3#\*}" "$1" "$2" | xargs grep -h "$1" | wc -l | tr -d ' '; }
fi

test "$(count_files '^gooo meta_operator_typechecker v1$' examples '*.gooo')" = "1"
test "$(count_files '^denominator ' examples '*.gooo')" = "1"
test "$(count_files '^precedence REFUTED>UNKNOWN>CLOSED$' examples '*.gooo')" = "1"
test "$(count_files '^unknown_fields stage,step,reason,unknown_class,next_operation,blocked_by$' examples '*.gooo')" = "1"
test "$(count_lines '^case ordinal=' examples '*.gooo')" = "7"
test "$(count_files '^// Code generated' internal '*.go')" -ge 1
