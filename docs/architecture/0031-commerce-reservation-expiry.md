# ADR-0031: Bounded commerce reservation expiry

- **Status:** Accepted
- **Date:** 2026-09-12
- **Related decisions:** [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0017](0017-commerce-consistency-and-payments.md), [ADR-0029](0029-commerce-customer-capability-boundary.md)

## Decision

An order reserves stock only while its payment status is `pending`. The active
Commerce module runs a bounded, once-per-minute releaser. It expires pending
orders older than fifteen minutes, atomically marks them `expired`/`cancelled`,
and restores the quantities recorded in immutable order lines. Processing is
limited to 100 orders per transaction cycle.

The conditional state transition claims the order before stock is restored;
therefore concurrent releasers and a late provider webhook cannot release the
same reservation twice. A webhook may only advance `pending` or `failed`
orders, so an expired order does not become paid without an explicit recovery
workflow.

## Consequences

The worker exists only while Commerce is enabled. No migration is required:
the existing immutable order lines and `created_at` field are sufficient. The
operation is portable SQL behind the Commerce repository and runs inside the
existing transaction boundary. During a blue/green deployment, both versions
continue to understand pending orders; the new version alone begins releasing
old holds. Provider sandbox verification and an operator-approved exception
flow for delayed payments remain separate production gates.
