# ADR-0159: SQLite external-write cache monitor

- Status: Accepted (bounded safety net; integration and soak open)
- Date: 2026-09-19
- Related decisions: [ADR-0155](0155-tenant-scoped-cache-api.md), [ADR-0158](0158-bounded-outbox-polling.md)

## Context

HydroGo invalidates its L1 cache when mutations pass through the application.
Operators may nevertheless update a SQLite file from a maintenance script or
DB tool. Without an observation path, those writes can leave stale values in
memory until restart.

## Decision

`internal/cache.SQLiteDataVersionMonitor` provides an opt-in, bounded watcher
around SQLite `PRAGMA data_version`. It polls at a validated base interval,
backs off on read errors, resets after a successful read, and invokes an
application-owned invalidation callback when another connection changes the
database version. Errors are reported through a callback and cancellation is
context-driven.

This is a safety net, not a replacement for application invalidation. SQLite
`data_version` is connection-relative: deployments must use a documented
stable connection policy (especially when a pool is present), and direct
database edits remain an operationally unsupported write path until an
integration/soak drill proves the chosen policy.

## Consequences

- External maintenance writes can trigger a bounded cache clear without a
  process restart.
- A broken or locked database cannot create an unbounded polling loop.
- The monitor adds periodic reads and should remain disabled when direct writes
  are forbidden or when the deployment has no stable connection policy.
- PostgreSQL and distributed invalidation require separate provider contracts;
  this ADR does not claim cross-dialect cache coherence.
