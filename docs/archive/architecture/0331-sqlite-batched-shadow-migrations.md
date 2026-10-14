# ADR-0331: Bounded SQLite shadow-table migrations

- **Status:** Accepted
- **Date:** 2026-09-29
- **Related:** [ADR-0002](0002-database-portability.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md)

## Decision

SQLite table rebuilds use a migration-manifest-driven shadow-table runner only
when the database is in WAL mode. The runner creates the replacement schema,
installs source-table triggers that mirror inserts, updates, and deletes, and
copies rows in bounded `BEGIN IMMEDIATE` transactions. A durable cursor and
descriptor checksum let the same migration resume after process interruption;
resumption with changed migration instructions fails closed. The final
`BEGIN IMMEDIATE` transaction removes the mirror triggers, swaps tables,
recreates declared indexes/triggers, checks foreign keys, and records the
migration ledger entry atomically.

This reduces the time each copy write-lock is held; it does not make SQLite a
multi-writer database or promise uninterrupted writes during cutover. Readers
can continue under WAL, while a writer may wait up to the configured SQLite
busy timeout. Deployments must schedule the migration in a controlled window
and monitor database/WAL free space.

## Compatibility and recovery

- Migration SQL/metadata is immutable after release. The SQLite descriptor is
  included in the runtime checksum and checked by `migrationguard`.
- A crash before cutover leaves the old table authoritative. Re-run the same
  migration to resume; do not manually remove its shadow table, state row, or
  triggers.
- A failed cutover rolls back as one SQLite transaction. Correct the migration
  before retrying; do not change a descriptor after a durable copy begins.
- The replacement schema must preserve all required columns and constraints.
  Required indexes and triggers must be declared in `after_swap`.
- The current runner rejects `WITHOUT ROWID` tables, a replacement table with
  an `INTEGER PRIMARY KEY`, and databases not using WAL. Such migrations use a
  separately reviewed maintenance procedure.
- Old and new binaries must both tolerate the resulting schema for the
  deployment's rollback window. The runner cannot certify application-level
  read/write compatibility; migration metadata and release preflight own that
  gate. In-place data-format conversion is not reversible by binary rollback.

## Capacity and verification

The migration temporarily needs disk space for a second copy of the target
table plus indexes and WAL growth. Operators must set a disk estimate in the
migration manifest and ensure free space before execution. Copy batches are
capped at 1000 rows (default 200); SQLite remains single-writer, so batch size
is a throughput/latency tradeoff. Compact runs one migration connection and no
background worker. Distributed deployments must not share a SQLite file across
application nodes.

Contract tests cover concurrent source writes being mirrored during backfill,
delete/update/insert parity, interrupted cutover rollback and restart,
foreign-key validation, index recreation, and unsupported table/database
forms. PostgreSQL continues to use its existing migration path; this decision
does not alter other dialect contracts.
