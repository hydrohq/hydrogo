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
The contract gate must exercise each target family, including migrations,
pool limits, transaction rollback, tenant isolation, repository behavior,
backup/restore, and schema inspection. Until that full matrix passes, MySQL is
available as an experimental adapter and is not described as certified.

## Operational impact

The driver uses Go's standard `database/sql` pool; MySQL connections receive a
bounded lifetime below common server idle timeouts. Existing profile pool
bounds remain in effect. MySQL migrations add no runtime workers or services.
Every migration gains a reviewed MySQL script and rollback/compatibility
metadata; old SQLite/PostgreSQL checksums and scripts remain immutable.

## Consequences

SQLite and PostgreSQL contracts remain unchanged. MySQL support expands the
release matrix and maintenance surface. The implementation may not be marked
certified or close #127 until the full repository contract suite has passed on
both MySQL 8 and MariaDB 10.5+.
