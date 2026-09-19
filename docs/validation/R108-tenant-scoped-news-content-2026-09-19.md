# R108 — Tenant-scoped news content and jobs

Date: 2026-09-19

## Evidence

News repository reads, lists, lifecycle writes and scheduled discovery now use
the validated tenant context when shared mode is enabled. Scheduled jobs carry
tenant IDs and the worker refuses to enqueue without scope; legacy payloads are
dead-lettered in shared mode.

Commands:

- `go test ./modules/news/... ./cmd/hydro -count=1`
- `go test ./migrations ./tests/contract -count=1`
- `go run ./tools/migrationguard`

Result: passed.

This is local SQLite/contract evidence. Existing-row backfill, queue-level
tenant predicates and PostgreSQL production certification remain open.
