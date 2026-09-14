# R5 validation: bounded list indexes

Date: 2026-09-14

- `go test ./internal/database` passes, including the SQLite `EXPLAIN QUERY PLAN`
  regression fixture for public page and news lists.
- `go run ./tools/migrationguard -root migrations` passes for 38 migrations.
- Page and news index migrations are module-scoped so filtered/inactive module
  runs do not reference absent tables.
- PostgreSQL planner selection, index size, and lock timing remain external
  release gates and are not claimed by this local validation.
