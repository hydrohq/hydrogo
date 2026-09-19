# ADR-0158: Bounded outbox polling lifecycle

- Status: Accepted (worker-loop foundation; production soak open)
- Date: 2026-09-19
- Related decisions: [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0019](0019-external-webhook-delivery.md)

## Context

Compact SQLite has no portable `LISTEN/NOTIFY` equivalent. A worker that polls
an outbox can either spin on an empty queue/locked database or become difficult
to stop and operate. The existing lease/claim contract must be wrapped in a
bounded lifecycle rather than requiring each deployment to invent a loop.

## Decision

`internal/webhooks.Worker.Run` provides the lifecycle. It polls `RunOnce`,
starts at a bounded base interval, exponentially backs off empty/transient-error
iterations up to a bounded maximum, resets to the base interval after a
successful claim, reports delivery/database errors through an optional callback,
and exits on context cancellation. Invalid intervals fail closed. The loop uses
no database-specific notification API, so it works with SQLite and PostgreSQL;
providers may add wake-up hints later without changing the lease contract.

## Consequences

- SQLite does not receive an unbounded tight `SELECT` loop during idle or lock
  periods, reducing WAL/disk contention risk.
- Lease ownership, retry scheduling and idempotent delivery remain in the
  existing store/dispatcher boundaries.
- Real lock/throughput soak, worker count, and PostgreSQL wake-up optimization
  remain deployment capacity gates.
