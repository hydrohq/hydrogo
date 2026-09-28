# R127 — MySQL adapter bootstrap

The non-Compact binary now registers `go-sql-driver/mysql`, opens the MySQL
driver through `database/sql`, applies the existing bounded pool defaults, and
sets a three-minute connection lifetime. Compact builds omit the driver and
fail closed if MySQL is selected. MySQL uses positional `?` parameters, and
the migration ledger uses InnoDB-compatible `VARCHAR`/`CHAR` key columns and
does not wrap its creation in a transaction.

The pool, rollback, migration ledger, migration lock, and multiple-statement
migration contracts passed against local MariaDB 13.0.2:

```text
HYDRO_TEST_MYSQL_URL=user:password@tcp(127.0.0.1:3306)/hydro_test go test ./internal/database -run 'TestMySQL' -count=1 -v
PASS
```

This is bootstrap evidence only, not the adapter certification required by
ADR-0283 or issue #127. The 87 existing migration bundles do not yet contain
MySQL scripts, and repository SQL, backup/recovery, tenant isolation, and the
full contract suite have not been certified on MySQL 8 or MariaDB 10.5+. Do not
enable MySQL for a production Edition until those gates pass.
