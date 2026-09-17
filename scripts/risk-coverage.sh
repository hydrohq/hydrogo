#!/usr/bin/env bash
set -euo pipefail

# Package-specific floors prevent a broad repository average from hiding
# regressions in security, member, payment, and GraphQL paths.
targets=(
  "internal/auth/sqlstore:4.0"
  "modules/members/transport:45.0"
  "modules/commerce/providers:43.0"
  "modules/api/graphql:65.0"
)
tmp_root="${TMPDIR:-/tmp}/hydrogo-risk-coverage-$$"
mkdir -p "$tmp_root"
trap 'rm -rf "$tmp_root"' EXIT

for target in "${targets[@]}"; do
  package="${target%%:*}"
  floor="${target##*:}"
  profile="$tmp_root/$(echo "$package" | tr '/' '_').out"
  echo "==> risk coverage $package (floor ${floor}%)"
  go test "./$package" -covermode=atomic -coverprofile="$profile"
  actual="$(go tool cover -func="$profile" | awk '/^total:/ {gsub("%", "", $3); print $3}')"
  if [[ -z "$actual" ]]; then
    echo "risk coverage: unable to read total for $package" >&2
    exit 1
  fi
  if ! awk -v actual="$actual" -v floor="$floor" 'BEGIN { exit !(actual + 0 >= floor + 0) }'; then
    echo "risk coverage: $package is ${actual}% (floor ${floor}%)" >&2
    exit 1
  fi
  echo "PASS: $package ${actual}% >= ${floor}%"
done

echo "PASS: risk-weighted coverage floors"
