# ADR-0275: Read-only live migration ledger diff

- **Status:** Accepted (migration ledger only; physical catalog comparison remains open)
- **Date:** 2026-09-28
- **Related:** [ADR-0092: Signed schema compatibility preflight](0092-signed-schema-compatibility-preflight.md), [ADR-0039: Conditional cross-module migrations](0039-conditional-cross-module-migrations.md)

## Context

Operators need a local check of the selected Edition's migration state against
the migration files shipped with the current binary. `hydro schema diff`
remains the signed release-to-release compatibility check defined by ADR-0092;
it does not inspect a live database.

## Decision

`hydro db diff` opens the configured SQLite or PostgreSQL database and calls
the existing read-only migration inspection contract. It reports selected
pending migration IDs, rejects unknown ledger rows and source checksum
mismatches, and never creates or mutates the ledger. JSON output is available
for automation. It does not inspect database tables or columns; that requires
a separately certified catalog contract for each database adapter.

The command uses the configured module selection, applies no migrations,
starts no worker, and creates no persistent files. The report is operator
output and must not be copied into customer-facing Admin Studio messages.

## Verification and operational impact

- SQLite contract tests prove pending migrations are reported without
  creating `schema_migrations`.
- PostgreSQL uses the existing `to_regclass` and ledger query behavior; no DDL
  or migration metadata is added. Adapter integration remains covered by the
  database contract environment.
- Runtime work is bounded by the migration bundle size and is a one-shot CLI
  operation; it adds no Compact profile memory, connection, worker, or cache
  overhead beyond opening the configured database connection.
- The command is safe to run during normal service operation because it only
  reads the migration ledger.

## Consequences

Operators get a live pending/checksum/unknown-migration preflight alongside
the existing signed manifest comparison. This does not detect manual table
or column edits. A future physical schema diff must define adapter-specific
catalog normalization and be certified for both SQLite and PostgreSQL before
the tool may claim physical drift detection.
