#!/usr/bin/env bash
set -euo pipefail

run_required() {
  local name="$1"
  shift
  printf '==> %s\n' "$name"
  if "$@"; then
    printf '%s\n' "- PASS: ${name}" >> "$report_path"
  else
    status=$?
    printf '%s\n' "- FAIL: ${name} (exit ${status})" >> "$report_path"
    return "$status"
  fi
}

gate_dir="${HYDRO_GATE_DIR:-.tmp/gates}"
mkdir -p "$gate_dir"
report_path="${HYDRO_GATE_REPORT:-${gate_dir}/production-gates-$(date -u +%Y%m%d-%H%M%S).md}"
commit="$(git rev-parse HEAD)"
profile="${HYDRO_CAPACITY_PROFILE:-compact (default)}"
go_version="$(go version)"
os="$(uname -srvmo)"
cat > "$report_path" <<EOF
# HydroGo production gate report

- Started: $(date -u --iso-8601=seconds)
- Commit: `$commit`
- Profile: `$profile`
- Toolchain: `$go_version`
- OS: `$os`
- PostgreSQL contract target: configured (credentials omitted)

## Automated gates
EOF

printf '%s\n' 'HydroGo production gate preflight'
printf '%s\n' 'This runner executes reproducible local/CI gates; it does not replace a real restore, proxy, key-ceremony or customer-capacity drill.'

if [[ -z "${HYDRO_TEST_POSTGRES_URL:-}" ]]; then
  printf '%s\n' 'HYDRO_TEST_POSTGRES_URL is required; PostgreSQL contract tests must not be skipped.' >&2
  exit 1
fi

run_required 'gofmt' bash -c '[[ -z "$(gofmt -l .)" ]]'
run_required 'module integrity' go mod verify
run_required 'workflow action pins' go run ./tools/actionguard
run_required 'dependency vulnerability scan' go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
run_required 'go vet' go vet ./...
run_required 'full test suite' go test ./... -count=1 -p 1 -timeout=5m
run_required 'shuffled test suite' go test ./... -shuffle=on -count=1 -p 1 -timeout=5m

run_required 'race-sensitive packages' go test -race -p 1 \
  ./internal/... ./modules/... ./cmd/hydro ./tests/contract -count=1 -timeout=5m

build_ldflags='-X main.buildCommit=production-gate -X main.buildTime=production-gate'
run_required 'reproducible hydro build' bash -c '
  set -euo pipefail
  go build -trimpath -buildvcs=false -ldflags "$1 -buildid=" -o "$2/hydro-a" ./cmd/hydro
  go build -trimpath -buildvcs=false -ldflags "$1 -buildid=" -o "$2/hydro-b" ./cmd/hydro
  if command -v sha256sum >/dev/null 2>&1; then
    a=$(sha256sum "$2/hydro-a" | cut -d" " -f1)
    b=$(sha256sum "$2/hydro-b" | cut -d" " -f1)
  elif command -v shasum >/dev/null 2>&1; then
    a=$(shasum -a 256 "$2/hydro-a" | cut -d" " -f1)
    b=$(shasum -a 256 "$2/hydro-b" | cut -d" " -f1)
  else
    printf '%s\n' 'sha256sum or shasum is required' >&2
    exit 1
  fi
  test "$a" = "$b"
' bash "$build_ldflags" "$gate_dir"

run_required 'portable benchmark' go test ./internal/portable -run '^$' -bench '^BenchmarkSQLitePortableRoundTrip256$' -benchtime=10x

printf '%s\n' 'PASS: automated gates completed.'
cat >> "$report_path" <<EOF

## External drills still required
- Real PostgreSQL restore into an empty target
- Reverse-proxy two-process blue/green restart and rollback
- External backup and signing-key recovery
- Production key ceremony and provenance publication
- Representative customer capacity, RPO and RTO measurements

Report completed: $(date -u --iso-8601=seconds)
EOF
printf 'Evidence report: %s\n' "$report_path"
printf '%s\n' 'REQUIRED EXTERNAL DRILLS: real PostgreSQL restore, reverse-proxy/process rollback, external backup/key recovery, production key ceremony, and customer RPO/RTO/capacity measurement.'
