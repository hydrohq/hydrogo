# R98 — SQLite cache watch runtime integration

Date: 2026-09-19

## Evidence

When explicitly enabled for SQLite, `serve` obtains a dedicated database
connection, binds `SQLiteDataVersionMonitor` to it, and clears the page cache on
an observed external database version change. The feature is absent for
PostgreSQL, disabled configuration, or a deployment without page caching.

Commands:

- `go test ./internal/config ./cmd/hydro ./internal/cache -count=1`
- `go vet ./internal/config ./cmd/hydro ./internal/cache`

Result: passed.

This is local wiring evidence only. A real external writer, stale response
integration test, lock/throughput soak, and multi-instance invalidation drill
remain open production gates.
