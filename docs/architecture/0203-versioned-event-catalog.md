# ADR-0203: Versioned event catalog

- **Status:** Accepted
- **Date:** 2026-09-21
- **Related:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0019](0019-external-webhook-delivery.md), [ADR-0163](0163-tenant-context-outbox-propagation.md), [ADR-0202](0202-core-client-edition-and-construction-pilot.md)

## Context

Outbox and signed webhook delivery already provide reliable transport, but a
free-form topic string cannot communicate compatibility, ownership, or PII
expectations to Client Editions and webhook consumers.

## Decision

Core publishes a versioned catalog. An event is identified by its topic and
integer payload version. New versions are additive catalog entries; changing a
published payload in place is forbidden. Payloads use opaque IDs and minimal
business state by default. An event marked PII requires an explicit consumer
contract and must not be sent to broad subscriptions.

The catalog is a boundary, not an in-process event bus. Delivery remains the
existing durable outbox, signed webhook, lease, retry, and idempotency path.
Client Editions may add their own reviewed namespaced events without placing
their domain vocabulary in Core.

## Validation

- Unit tests reject unknown topics and versions.
- A compatibility helper rejects removal or PII reclassification of a
  published topic/version pair; additions require a new explicit entry.
- Event producers and webhook consumers must use a catalog entry before a
  public subscription is documented.
- Payload compatibility, tenant scope, idempotency, and PII minimization are
  reviewed with the producer change.
