# ADR-0277: Read-only SQLite physical schema diff

- **Status:** Accepted for SQLite; PostgreSQL and MySQL catalog adapters remain open
- **Date:** 2026-09-29
- **Supersedes:** The SQLite scope of the physical-catalog deferral in [ADR-0275](0275-live-migration-ledger-diff.md)
- **Related:** [ADR-0006](0006-extensible-database-adapters.md)

## Context

The migration-ledger comparison in ADR-0275 could not detect manual changes to
physical objects. SQLite editions need a read-only check against the same
module-selected migration bundle that would build a new database.

## Decision

For SQLite, `hydro db diff` builds a temporary in-memory reference database by
applying the selected migration bundle, then compares `sqlite_master` object
kind, name, and normalized SQL definition with the configured live database.
It reports missing objects, unexpected objects, and changed definitions for
tables, indexes, views, and triggers. The configured database is only queried;
the inspector never creates or changes its migration ledger.

PostgreSQL catalog normalization is not certified by this decision. MySQL is
not a certified HydroGo dialect under ADR-0006. Neither is reported as having a
physical schema match. The existing ledger comparison remains available for
those supported database paths.

## Verification and operational impact

Tests prove missing index and unexpected table detection, migration-reference
construction, and that a live database without a migration ledger stays
unchanged. The comparison is deterministic and bounded by the migration
bundle and catalog size. It opens no worker, writes no persistent reference
database, and adds no steady-state connections or memory to Compact runtime;
the temporary catalog exists only for this one-shot CLI command.

SQLite stores the executed DDL in `sqlite_master`, so exact logical object
definitions are comparable against a reference created by the same SQLite
engine. This does not claim semantic analysis of data, query plans, collations,
or database pragmas. PostgreSQL and MySQL support require their own catalog
normalization and certified contract suites before physical drift can be
reported for those engines.
