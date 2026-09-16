# HTTP idempotency guide

## What exists today

Idempotency is already implemented where duplicate side effects are most
dangerous:

- Commerce checkout uses `Idempotency-Key` plus a unique `(user_id, key)` order
  constraint and rejects a reused key with a different order body.
- Payment webhooks and provider requests use durable event/provider keys.
- Outbox/webhook dispatch, jobs, news scheduling, erasure replay and SDK install
  have their own bounded replay or lease contracts.

These are intentionally domain-specific. They are not a promise that every HTTP
mutation is globally exactly-once.

The first bounded HTTP pilot is enabled for `POST /forms/{form}/submit`.
It accepts a visible-ASCII `Idempotency-Key` of 16–128 bytes. A matching retry
returns `202 Accepted` without creating a second inbox row; reusing the key
with a different payload returns `409 Conflict`. The key and request hash are
durable in both SQLite and PostgreSQL. Calls without a key remain compatible.

## Proposed common layer

For a route that opts in to ADR-0136, clients should send:

```http
Idempotency-Key: checkout-client-20260917-7e2d...
```

The key must be unique for the actor/project and one logical operation. Retrying
the same request replays the original response. Reusing the key for a different
body is a conflict and must not reach the application service.

The implementation order is:

1. Add the SQLite/PostgreSQL table and atomic claim/fingerprint contract.
2. Integrate one non-payment mutation and certify replay, conflict, pending,
   expiry and crash-after-commit behavior.
3. Add generated client helpers and OpenAPI header/error documentation.
4. Expand only to routes whose side effects and response replay semantics are
   explicitly reviewed.

Idempotency records are bounded operational data. They must have a retention
policy, redaction rules, metrics and an operator purge path. Provider-specific
payment/webhook guarantees and at-least-once email delivery remain separate.
