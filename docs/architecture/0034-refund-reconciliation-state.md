# ADR-0034: Refund reconciliation state

- **Status:** Accepted
- **Date:** 2026-09-12
- **Related decisions:** [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0017](0017-commerce-consistency-and-payments.md), [ADR-0032](0032-payment-provider-transport-safety.md)

## Decision

Before a provider refund request is made, the corresponding paid transaction is
atomically moved to `refund_pending`. A second request cannot issue another
provider refund. A confirmed provider success transitions this state to
`refunded`; an ambiguous provider or subsequent database failure remains
`refund_pending` for operator reconciliation.

## Consequences

The safe automatic behavior is to avoid duplicate financial effects. An
operator must verify a pending refund against the provider before resolving it;
the system does not guess whether a timed-out provider call succeeded. This
uses the existing status column, needs no migration, and preserves old paid and
refunded rows during a blue/green deployment.
