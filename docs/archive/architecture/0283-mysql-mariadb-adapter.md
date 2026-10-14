# ADR-0283: MySQL and MariaDB adapter certification

- **Status:** Accepted
- **Date:** 2026-09-28
- **Supersedes:** The MySQL deferral in [ADR-0006](0006-extensible-database-adapters.md)
- **Related:** [ADR-0282](0282-postgres-physical-schema-diff.md)

## Context

Issue #127 establishes demand for cPanel and enterprise MySQL/MariaDB
deployments. Adding a connection driver alone does not certify the product:
module queries, migration semantics, tenant behavior, recovery, and backup
contracts must work without leaking dialect handling into domain services.

## Decision

Add `database.MySQL` behind the existing `database/sql` boundary using
`go-sql-driver/mysql`. Keep MySQL-specific connection behavior and query
selection inside `internal/database` and repository implementations. MySQL
uses question-mark bind parameters, application-generated identifiers, and
engine-appropriate migration scripts. DDL migrations are treated as
non-transactional unless a migration explicitly proves otherwise. Compact
builds omit the MySQL driver just as they omit PostgreSQL.

The certification target is MySQL 8.0+ and MariaDB 10.5+ as stated in #127.
The contract gate exercises each target family, including migrations, pool
limits, transaction rollback, tenant isolation, repository behavior,
backup/restore, and schema inspection. MySQL remains experimental until the
evidence below records a passing full matrix.

## Operational impact

The driver uses Go's standard `database/sql` pool; MySQL connections receive a
bounded lifetime below common server idle timeouts. Existing profile pool
bounds remain in effect. MySQL migrations add no runtime workers or services.
Every migration gains a reviewed MySQL script and rollback/compatibility
metadata; old SQLite/PostgreSQL checksums and scripts remain immutable.

## Consequences

SQLite and PostgreSQL contracts remain unchanged. MySQL support expands the
release matrix and maintenance surface. #127 may close only after the full
repository contract suite passes on both MySQL 8 and MariaDB 10.5+; the current
certification evidence is recorded below.

## Certification evidence

**Certified 2026-09-28:** `go test ./... -count=1` passed on MySQL 8.0.46 and
MariaDB 10.5.29 using the matching vendor dump and restore clients. The suite
includes bounded pool behavior, transaction rollback, repository tenant
contracts, applied migration checksums, and MySQL physical schema drift
inspection. `internal/backup` also completed a full snapshot and staging
restore on each engine, verified restored content and media, re-applied restore
quarantine, and confirmed both immutable audit triggers survived the round
trip. `go run ./tools/migrationguard` passed with 88 migrations.
