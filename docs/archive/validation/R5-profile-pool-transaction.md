# R5 validation: profile pool and transaction contract

## Local evidence

- `go test ./internal/database ./internal/capacity`
- `TestCapacityProfilesMapToBoundedDatabasePools` covers Compact, Standard,
  and Distributed budgets.
- `TestTransactorCommitsAndRollsBackAtomically` proves commit-on-success and
  rollback-on-callback-error with SQLite.

## Open external gate

PostgreSQL lock contention and statement timeout behavior still require the
supported PostgreSQL service and deployment configuration. This local check
does not claim that production gate is complete.
