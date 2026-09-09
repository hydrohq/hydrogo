#!/usr/bin/env bash
set -euo pipefail

run_required() {
  local name="$1"
  shift
  printf '==> %s\n' "$name"
  "$@"
}

printf '%s\n' 'HydroGo production gate preflight'
printf '%s\n' 'This runner executes reproducible local/CI gates; it does not replace a real restore, proxy, key-ceremony or customer-capacity drill.'

if [[ -z "${HYDRO_TEST_POSTGRES_URL:-}" ]]; then
  printf '%s\n' 'HYDRO_TEST_POSTGRES_URL is required; PostgreSQL contract tests must not be skipped.' >&2
  exit 1
fi

run_required 'gofmt' bash -c '[[ -z "$(gofmt -l .)" ]]'
run_required 'module integrity' go mod verify
run_required 'dependency vulnerability scan' go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
run_required 'go vet' go vet ./...
run_required 'full test suite' go test ./... -count=1 -p 1
run_required 'shuffled test suite' go test ./... -shuffle=on -count=1 -p 1

run_required 'race-sensitive packages' go test -race -p 1 \
  ./internal/cache ./internal/release ./internal/database ./internal/backup \
  ./internal/security ./internal/server ./internal/jobs ./internal/jobs/sqlstore \
  ./internal/auth ./internal/media ./internal/mailer ./internal/portable \
  ./modules/news ./modules/site ./cmd/hydro -count=1

gate_dir=.tmp/gates
mkdir -p "$gate_dir"
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
printf '%s\n' 'REQUIRED EXTERNAL DRILLS: real PostgreSQL restore, reverse-proxy/process rollback, external backup/key recovery, production key ceremony, and customer RPO/RTO/capacity measurement.'
