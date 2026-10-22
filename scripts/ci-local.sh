#!/usr/bin/env bash
# Runs jobs of .github/workflows/ci.yml on this machine with nektos/act, so the
# checks can be run without GitHub Actions: scripts/ci-local.sh [job ...]
# Default jobs: go first-run sveltekit-starter crm-starter.
#
# Needs Docker and act (go install github.com/nektos/act@latest; set ACT to its
# path if it is not on PATH). The workflow is copied to a temporary file first:
#  - the PostgreSQL and Redis service ports move to 15432 and 16379, so a
#    database or cache already listening on this machine is left alone;
#  - upload-artifact steps are dropped (act has no compatible artifact server).
# Not runnable here: the container image build (arm64/QEMU), the Windows jobs.
set -euo pipefail

cd "$(dirname "$0")/.."
act_bin="${ACT:-act}"
command -v "$act_bin" >/dev/null || { echo "act not found: go install github.com/nektos/act@latest (or set ACT)" >&2; exit 1; }
command -v docker >/dev/null || { echo "docker is required" >&2; exit 1; }

jobs=("$@")
[ ${#jobs[@]} -gt 0 ] || jobs=(go first-run sveltekit-starter crm-starter)

workflow="$(mktemp --suffix=.yml)"
trap 'rm -f "$workflow"' EXIT
sed -e 's#- 5432:5432#- 15432:5432#; s#- 6379:6379#- 16379:6379#' \
    -e 's#127\.0\.0\.1:5432#127.0.0.1:15432#g; s#127\.0\.0\.1:6379#127.0.0.1:16379#g' \
    -e 's#^\( *\)PGPASSWORD: \(.*\)$#&\n\1PGPORT: "15432"#' \
    .github/workflows/ci.yml |
  awk '
    function flush() { if (step != "" && step !~ /actions\/upload-artifact/) printf "%s", step; step = "" }
    /^      - / { flush(); step = $0 "\n"; next }
    /^ {0,6}[^ ]/ { flush(); print; next }
    step != "" { step = step $0 "\n"; next }
    { print }
    END { flush() }
  ' >"$workflow"

status=0
for job in "${jobs[@]}"; do
  echo "==> $job"
  if ! "$act_bin" pull_request \
      -P ubuntu-latest=ghcr.io/catthehacker/ubuntu:act-latest \
      --container-architecture linux/amd64 --network host \
      -W "$workflow" -j "$job"; then
    echo "FAIL: $job" >&2
    status=1
  fi
done
[ "$status" -eq 0 ] && echo "PASS: ${jobs[*]}"
exit "$status"
