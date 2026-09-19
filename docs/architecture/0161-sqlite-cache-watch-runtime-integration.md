# ADR-0161: Opt-in SQLite cache watch runtime integration

- Status: Accepted (single-process integration; external-write soak open)
- Date: 2026-09-19
- Related decisions: [ADR-0159](0159-sqlite-external-write-cache-monitor.md), [ADR-0158](0158-bounded-outbox-polling.md)

## Context

The SQLite `data_version` watcher must be connected to the actual public page
cache to help with operator scripts that bypass application invalidation. A
pooled `*sql.DB` is not sufficient because `data_version` is connection-relative
and may otherwise observe different connections.

## Decision

`database.sqlite_external_write_watch` (or
`HYDRO_SQLITE_EXTERNAL_WRITE_WATCH=true`) opt-in opens one dedicated SQLite
connection for the process lifetime. The bounded watcher reads `PRAGMA
data_version` through that connection and clears the configured page cache on
external changes. Read errors report to stderr and back off; cancellation closes
the dedicated connection. PostgreSQL and disabled/empty-page-cache profiles do
not create a connection or goroutine.

## Consequences

- Compact operators can explicitly enable stale-cache protection for supported
  maintenance workflows.
- The feature has measurable resource cost and is not silently enabled.
- Dedicated-connection behavior, external writer integration, lock contention,
  and multi-instance cache invalidation still require deployment evidence.
