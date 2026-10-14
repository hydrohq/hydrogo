# R127 — MySQL adapter implementation status

The regular build registers `go-sql-driver/mysql` through `database/sql`, applies
bounded pool defaults, and gives connections a bounded lifetime. Compact builds
omit the driver and fail closed if MySQL is selected. MySQL uses positional `?`
parameters and an InnoDB-compatible migration ledger.

All 87 embedded migration bundles now include MySQL scripts and metadata. The
complete migration chain, tenant-scoped legacy data backfill, representative
foreign keys and constraints, portfolio tag backfill, pool limits, rollback,
migration runner, schema diff, and scratch database cleanup passed against MySQL 8.0.46, MariaDB 10.5, and MariaDB 13.0.2. The MySQL
integration contracts passed on each version, and the full `go test ./...`
suite passed with `HYDRO_TEST_MYSQL_URL` configured for MySQL 8.0.46 and
MariaDB 10.5. `go run ./tools/migrationguard` also passes.

The adapter is still experimental pending the complete repository contract
suite running its database behavior on MySQL (the existing package tests still
primarily exercise SQLite/PostgreSQL), backup/restore contracts, and tenant
isolation across repository families. Keep issue #127 open until those gates
pass.
