# R5 validation: module application boundary guard

Date: 2026-09-14

- `go test ./tools/importguard` passes with violating service fixtures and
  adapter/test exclusions, including raw SQL, SQLite, PostgreSQL, Redis, and
  durable outbox imports.
- The guard scans non-test module application/domain packages and rejects
  concrete SQL/cache/queue implementation imports.
- PostgreSQL/Redis runtime behavior remains covered by their respective
  integration and external release gates.
