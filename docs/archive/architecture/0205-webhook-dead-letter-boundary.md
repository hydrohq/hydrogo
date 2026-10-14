# ADR-0205: Bounded Webhook Delivery and Dead-Letter Boundary

Status: Accepted  
Date: 2026-09-21

## Context

Webhook delivery already used signed HTTP requests and bounded per-request retries, but a permanently unavailable consumer could remain in the durable outbox indefinitely. This created unbounded operator noise and made recovery state implicit.

## Decision

The outbox worker uses a bounded delivery-attempt threshold (default eight, configurable for a deployment). Once an event reaches the threshold, the leased event is atomically archived into `outbox_dead_letters` and removed from the active queue. The archive retains tenant, topic, aggregate, payload version, attempt count, failure reason, and timestamp. Older binaries remain compatible because the migration is additive; dead-letter activation requires the new worker and migration ledger state.

## Consequences

- Operators get a durable, queryable failure surface instead of infinite retries.
- Recovery tooling can be added without replaying active traffic accidentally.
- The archive is append-only and must follow retention/PII controls.
- Deployments must apply the expand migration before enabling the bounded worker.
