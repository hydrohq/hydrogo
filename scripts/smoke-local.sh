#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "${repository_root}/.tmp"
smoke_root="$(mktemp -d "${repository_root}/.tmp/smoke-linux.XXXXXX")"
binary="${smoke_root}/hydro"
database="${smoke_root}/hydro.sqlite"
server_log="${smoke_root}/serve.log"
server_pid=""

# Compact smoke is intentionally independent of any developer Redis setup.
unset HYDRO_REDIS_URL HYDRO_REDIS_KEY_PREFIX

cleanup() {
  if [[ -n "${server_pid}" ]] && kill -0 "${server_pid}" 2>/dev/null; then
    kill "${server_pid}" 2>/dev/null || true
    wait "${server_pid}" 2>/dev/null || true
  fi
  if [[ -s "${server_log}" ]]; then
    printf '%s\n' 'smoke server log:'
    cat "${server_log}"
  fi
  rm -rf "${smoke_root}"
}
trap cleanup EXIT

printf '%s\n' '==> build smoke binary'
go build -trimpath -buildvcs=false -ldflags '-X main.buildCommit=smoke -X main.buildTime=smoke' -o "${binary}" "${repository_root}/cmd/hydro"

printf '%s\n' '==> migrate isolated SQLite database'
"${binary}" migrate --database-dialect sqlite --database-dsn "${database}"

port=$((18000 + RANDOM % 1000))
address="127.0.0.1:${port}"
printf '==> start isolated server on %s\n' "${address}"
"${binary}" serve --database-dialect sqlite --database-dsn "${database}" --address "${address}" >"${server_log}" 2>&1 &
server_pid=$!

ready=0
for _ in $(seq 1 40); do
  if ! kill -0 "${server_pid}" 2>/dev/null; then
    cat "${server_log}"
    exit 1
  fi
  if curl --fail --silent --show-error "http://${address}/healthz" >/dev/null && \
     curl --fail --silent --show-error "http://${address}/readyz" | grep -q 'ready'; then
    ready=1
    break
  fi
  sleep 0.25
done

if [[ "${ready}" -ne 1 ]]; then
  printf '%s\n' 'health/readiness smoke check timed out' >&2
  exit 1
fi
printf '%s\n' 'PASS: healthz and readyz returned 200 from the built binary.'

printf '%s\n' '==> graceful shutdown'
kill -TERM "${server_pid}"
if wait "${server_pid}"; then
  server_pid=""
else
  exit_code=$?
  printf 'server exited with code %d during graceful shutdown\n' "${exit_code}" >&2
  exit "${exit_code}"
fi
printf '%s\n' 'PASS: server stopped cleanly after SIGTERM.'
