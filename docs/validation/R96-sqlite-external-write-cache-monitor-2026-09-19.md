# R96 — SQLite external-write cache monitor

Date: 2026-09-19

## Evidence

`internal/cache.SQLiteDataVersionMonitor` watches SQLite `PRAGMA data_version`
with bounded polling and exponential backoff on read errors. The contract test
proves an external version change invokes invalidation and that a locked/read
error sequence backs off at 100ms → 200ms → 400ms. Invalid bounds fail closed.

Commands:

- `go test ./internal/cache -run TestSQLiteDataVersionMonitor -count=1`
- `go vet ./internal/cache`

Result: passed.

This is repository-level behavior evidence only. A dedicated SQLite connection,
external writer, stale-cache integration test, and lock/throughput soak remain
open deployment gates; the monitor must not be presented as production cache
  coherence certification.
