# ADR-0032: Payment provider transport safety

- **Status:** Accepted
- **Date:** 2026-09-12
- **Related decisions:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0017](0017-commerce-consistency-and-payments.md)

## Decision

Payment provider adapters use a shared bounded HTTP transport: ten-second
timeout, no implicit redirect following, and a one-mebibyte response/body
limit. Provider response bodies and tokens are never included in domain errors.
Webhook money values are converted from a decimal representation to minor units
without a binary floating-point conversion.

## Consequences

The transport policy adds no process, worker, cache, or schema cost and is
fully inactive when Commerce is absent. Existing configured custom clients keep
an explicit redirect policy if supplied; otherwise the fail-closed policy is
applied. The boundary remains provider-neutral and does not replace required
sandbox certification for live accounts.
