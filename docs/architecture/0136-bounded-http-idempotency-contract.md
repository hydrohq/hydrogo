# ADR-0136: Bounded HTTP idempotency contract

- **Status:** Proposed
- **Date:** 2026-09-17
- **Related decisions:** [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0015](0015-versioned-http-api.md), [ADR-0017](0017-commerce-consistency-and-payments.md)
- **Implementation status:** Existing Commerce, payment-event, job, outbox, erasure and SDK idempotency remains active. The first bounded HTTP pilot is wired to Site form submissions; the universal mutation layer remains a staged rollout.

## Context

HydroGo already protects the highest-risk commerce and delivery flows with
domain-specific idempotency. Other retryable HTTP mutations (content, site
settings, forms, membership/admin actions and generated CRUD) still rely on
optimistic locking or endpoint-specific behavior. Mobile retries, browser
double-submit, proxy retries and ambiguous network failures can therefore be
handled inconsistently.

## Decision proposal

Add an optional, durable Core contract for HTTP mutations. It must be explicitly
enabled per route/module and must not replace domain idempotency where a domain
already has a stronger invariant.

- Apply only to `POST`, `PATCH` and `DELETE` mutation operations; reads are never
  deduplicated.
- Require a bounded `Idempotency-Key` (16–128 visible ASCII characters).
- Scope the key by project/tenant, authenticated actor (or an explicit
  anonymous-session identity), HTTP operation and normalized route.
- Store a request fingerprint (method, route, content type and canonical body),
  state (`pending`, `succeeded`, `failed`), status code, bounded response body
  and expiry timestamp in the database.
- Atomically claim the first request. A matching completed request replays the
  stored response; a different fingerprint returns `409 Conflict`; a concurrent
  pending request returns a bounded retry response rather than executing twice.
- Bound response size and retention. Sensitive headers, cookies, secrets and
  personal data must not be persisted in the replay record.
- Record audit/metrics at the developer/operator tier only. End users receive
  stable, non-technical conflict/retry messages.
- SQLite and PostgreSQL adapters must expose the same uniqueness, transaction,
  expiry and conflict behavior. Redis is not a correctness prerequisite.
- External providers remain at-least-once unless their own idempotency contract
  is available; payment and email reconciliation remain separate.

## Rollout and compatibility

The first implementation targets the non-payment form write flow. It is
backward-compatible when no key is supplied and uses the Site migration's
durable partial unique index. Expansion should follow SQLite/PostgreSQL
contract tests and response-replay observations. Existing commerce keys and
database constraints remain source-of-truth and are not migrated to this table.
Unknown or malformed keys fail closed before business logic executes. During
blue/green deployment, the table and record format must be readable by both
old and new binaries before the feature is enabled.

## Open gates

The ADR remains Proposed until a migration, certified dialect tests, bounded
response policy, route integration, replay/conflict metrics and rollback impact
documentation are delivered together.
