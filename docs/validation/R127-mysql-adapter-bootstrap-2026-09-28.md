# R127 — MySQL adapter implementation status

The regular build registers `go-sql-driver/mysql` through `database/sql`, applies
bounded pool defaults, and gives connections a bounded lifetime. Compact builds
omit the driver and fail closed if MySQL is selected. MySQL uses positional `?`
parameters and an InnoDB-compatible migration ledger.

All 87 embedded migration bundles now include MySQL scripts and metadata. The
complete migration chain, tenant-scoped legacy data backfill, representative
foreign keys and constraints, portfolio tag backfill, pool limits, rollback,
migration runner, schema diff, and scratch database cleanup passed against
MariaDB 13.0.2:

```text
HYDRO_TEST_MYSQL_URL=... go test ./internal/database -run '^TestMySQL' -count=1 -v
PASS
```

`go test ./...` and `go run ./tools/migrationguard` also pass. This remains
implementation evidence rather than full adapter certification. The complete
repository contract suite, backup/restore, tenant isolation across repository
families, MySQL 8.0+, and MariaDB 10.5+ matrix still need certification before
production support is promised. Keep issue #127 open until those gates pass.
