#!/usr/bin/env bash
set -euo pipefail

patterns=(
  'BEGIN (RSA|EC|OPENSSH|PRIVATE) KEY'
  'AKIA[0-9A-Z]{16}'
  'gh[pousr]_[A-Za-z0-9_]{20,}'
  'sk_live_[A-Za-z0-9]{16,}'
  'rk_live_[A-Za-z0-9]{16,}'
  'xox[baprs]-[A-Za-z0-9-]{20,}'
)
args=()
for pattern in "${patterns[@]}"; do args+=( -e "$pattern" ); done
if git grep -n -E "${args[@]}" -- . ':(exclude)security-report'; then
  echo 'credential pattern found in tracked files' >&2
  exit 1
fi
echo 'PASS: no high-confidence credentials found in tracked files'
