#!/usr/bin/env bash
set -euo pipefail

runs="${HYDRO_TEST_STABILITY_RUNS:-3}"
output="${HYDRO_TEST_STABILITY_REPORT:-.tmp/test-stability.json}"

if ! [[ "$runs" =~ ^[1-9][0-9]*$ ]] || (( runs > 20 )); then
  echo "HYDRO_TEST_STABILITY_RUNS must be an integer from 1 to 20" >&2
  exit 2
fi

mkdir -p "$(dirname "$output")"
targets=(./internal/auth/... ./internal/cache/... ./internal/database/... ./internal/security/... ./modules/... ./tests/contract)
started="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
passed=0
failed=0

for ((run = 1; run <= runs; run++)); do
  echo "==> stability run ${run}/${runs}"
  if go test -shuffle=on -count=1 -p 1 -timeout=5m "${targets[@]}"; then
    ((passed += 1))
  else
    ((failed += 1))
  fi
done

commit="$(git rev-parse HEAD)"
printf '{\n  "schema_version": 1,\n  "commit": "%s",\n  "started_at": "%s",\n  "runs": %d,\n  "passed": %d,\n  "failed": %d\n}\n' "$commit" "$started" "$runs" "$passed" "$failed" > "$output"

if (( failed > 0 )); then
  echo "FAIL: ${failed}/${runs} shuffled stability runs failed; report: $output" >&2
  exit 1
fi
echo "PASS: ${runs}/${runs} shuffled stability runs passed; report: $output"
