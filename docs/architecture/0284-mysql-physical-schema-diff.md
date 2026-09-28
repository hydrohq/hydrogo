# ADR-0284: MySQL physical schema diff

- **Status:** Accepted
- **Date:** 2026-09-28
- **Related:** [ADR-0282](0282-postgres-physical-schema-diff.md),
  [ADR-0283](0283-mysql-mariadb-adapter.md)

## Context

Issue #129 requires physical drift detection for each supported dialect.
MySQL and MariaDB expose their live structure through
`information_schema`, but unlike SQLite they cannot build an isolated
reference catalog in memory.

## Decision

The MySQL inspector acquires one connection, records its selected database,
creates a random scratch database, applies only the selected embedded MySQL
migrations there, compares tables, views, columns, indexes, and foreign keys,
restores the original selected database, and drops the scratch database. It
never reads or copies application rows from the inspected database. A failed
cleanup is returned as an error.

The configured role must have `CREATE` and `DROP` database privileges and
access to objects it creates in the scratch database. Operators without those
privileges can still use migration-ledger checks; physical inspection reports
the permission failure. The operation adds no persistent worker, connection,
or data. A scratch database is bounded to one per invocation and uses the
embedded migration bundle.

MySQL physical inspection is implemented and contract-tested independently
from adapter certification. The adapter remains experimental under ADR-0283
until its repository, backup/recovery, and supported-version test matrix is
complete. Catalog comparison covers logical object definitions; it does not
inspect data, grants, storage configuration, or query plans.

## Verification

The MySQL integration contract applies all 87 migration bundles, compares a
migration-built reference with an empty dedicated live database containing a
changed column type and missing index/foreign key, and verifies scratch
schemas are removed. The suite passed against MySQL 8.0.46, MariaDB 10.5, and MariaDB 13.0.2. It requires
`HYDRO_TEST_MYSQL_URL` and a test account with the scratch database privileges
described above. MySQL 8 and MariaDB minimum version coverage remain adapter
release gates under ADR-0283.
