#!/usr/bin/env bash
set -euo pipefail

command -v go >/dev/null || { echo 'go is required' >&2; exit 1; }
command -v psql >/dev/null || { echo 'postgresql-client is required' >&2; exit 1; }
command -v redis-cli >/dev/null || { echo 'redis-cli is required' >&2; exit 1; }

if command -v service >/dev/null 2>&1; then
  sudo -n service redis-server start >/dev/null 2>&1 || true
  if command -v pg_isready >/dev/null 2>&1; then
    sudo -n service postgresql start >/dev/null 2>&1 || true
  fi
fi

echo "Go: $(go version)"
echo "Redis: $(redis-cli ping 2>/dev/null || echo unavailable)"
if command -v pg_isready >/dev/null 2>&1; then
  echo "PostgreSQL: $(pg_isready 2>/dev/null || true)"
fi
echo 'WSL bootstrap check completed.'
