#!/usr/bin/env bash
set -euo pipefail

test "$(git status --porcelain)" = ""
test "$(rg -l '^gooo meta_operator_typechecker v1$' examples --glob '*.gooo' | wc -l | tr -d ' ')" = "1"
test "$(rg -l '^denominator ' examples --glob '*.gooo' | wc -l | tr -d ' ')" = "1"
test "$(rg -l '^precedence REFUTED>UNKNOWN>CLOSED$' examples --glob '*.gooo' | wc -l | tr -d ' ')" = "1"
test "$(rg -l '^unknown_fields stage,step,reason,unknown_class,next_operation,blocked_by$' examples --glob '*.gooo' | wc -l | tr -d ' ')" = "1"
test "$(rg '^case ordinal=' examples --glob '*.gooo' | wc -l | tr -d ' ')" = "7"
test "$(rg -l '^// Code generated' internal --glob '*.go' | wc -l | tr -d ' ')" -ge 1
