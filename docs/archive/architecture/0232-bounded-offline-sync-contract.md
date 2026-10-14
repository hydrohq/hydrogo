# ADR-0232: Bounded offline sync contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0136](0136-bounded-http-idempotency-contract.md), [ADR-0230](0230-bounded-relation-plan-contract.md)

## Decision

Offline-capable Editions exchange a bounded sync batch containing a cursor and
idempotent commands. Command IDs, types and idempotency keys are validated;
payload bytes, cursor size and batch count are capped and commands are ordered
deterministically before dispatch.

The contract does not decide conflict resolution. An Edition declares whether a
resource rejects stale base versions, merges a domain-specific patch or sends
the record to manual review. Attachments and event replay must use the same
command identity and durable outbox/ack boundary; Core never silently drops a
command or performs unbounded replay.

## Verification

Public tests cover duplicate IDs, ordering and payload/identity budgets.
Construction and field workflows must add tenant ownership, attachment retry,
cursor replay and conflict-policy tests on both supported database adapters.
