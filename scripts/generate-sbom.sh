#!/usr/bin/env bash
set -euo pipefail

output="sbom.spdx.json"
if [[ "${1:-}" == "-o" || "${1:-}" == "--output" ]]; then
  if [[ -z "${2:-}" ]]; then
    echo "usage: $0 [--output PATH]" >&2
    exit 2
  fi
  output="$2"
elif [[ -n "${1:-}" ]]; then
  echo "usage: $0 [--output PATH]" >&2
  exit 2
fi

go run ./tools/sbom -output "$output"
echo "SBOM written to $output"
