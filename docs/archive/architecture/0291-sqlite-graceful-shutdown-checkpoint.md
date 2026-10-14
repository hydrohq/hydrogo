# ADR-0291: SQLite graceful shutdown checkpoint

**Status:** Accepted  
**Date:** 2026-09-29  
**Related:** [ADR-0006](0006-extensible-database-adapters.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0190](0190-capacity-profiles-concurrency-budgets.md), [ADR-0271](0271-security-hardening-and-recovery.md)

## Context

SQLite WAL mode can leave committed frames in the write-ahead log when the
process stops. SQLite recovers those frames safely on the next open, but a
controlled runtime shutdown should flush and truncate them after requests and
database-backed queues have stopped. Startup recovery must continue to leave
WAL/SHM ownership to SQLite and must never unlink sidecars.

## Decision

- Runtime cleanup cancels the runtime context, then closes deferred application
  queues and resources in normal reverse-registration order.
- Before closing the SQLite pool, runtime issues `PRAGMA wal_checkpoint(TRUNCATE)`
  with the profile's bounded shutdown timeout. A busy or failed checkpoint is
  returned as a shutdown error; the database pool is still closed.
- The checkpoint is not run for PostgreSQL, CGI per-request execution, or other
  database operations outside the persistent `serve` lifecycle.
- SQLite remains responsible for WAL/SHM sidecar lifecycle. HydroGo does not
  delete those files.

## Compatibility and operations

This changes no schema, migration, or SQL domain contract. It uses the existing
SQLite adapter behavior and each profile's established shutdown budget. If the
deadline expires or another connection prevents truncation, the process reports
the checkpoint failure and still closes the pool; SQLite startup recovery
validates the database and performs its existing restart checkpoint. This adds
one bounded shutdown operation and no worker, connection, cache, or persistent
resource.

## Verification

The SQLite database contract test creates WAL frames, verifies the shutdown
checkpoint truncates them, and confirms committed rows remain readable. The
full application suite validates the existing PostgreSQL contract remains
unchanged.
