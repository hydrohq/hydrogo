# R129 — Physical schema drift inspection

`hydro db diff` now compares the live physical schema with the selected embedded
migration state for SQLite, PostgreSQL, and MySQL. The inspectors compare
logical tables, views, columns, indexes, and foreign keys. Inspection remains
read-only against the live schema; PostgreSQL and MySQL use isolated temporary
schemas/databases and remove them before returning.

Verification:

- `go test ./...` passed.
- `HYDRO_TEST_MYSQL_URL=... go test ./internal/database -run '^TestMySQL' -count=1 -v` passed against MySQL 8.0.46, MariaDB 10.5, and MariaDB 13.0.2.
- SQLite and PostgreSQL schema drift contract tests are included in `go test ./...`; PostgreSQL tests run when `HYDRO_TEST_POSTGRES_URL` is configured.

MySQL physical inspection needs `CREATE` and `DROP DATABASE` privileges for its
temporary reference schema. It compares logical schema definitions and does not
inspect data, grants, storage settings, or query plans. MySQL adapter
certification is tracked separately under ADR-0283 and issue #127.
