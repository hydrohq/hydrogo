# ADR-0131: Durable aggregate analytics with explicit consent

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0125](0125-versioned-analytics-consent-contract.md), [ADR-0022](0022-consent-based-product-analytics-and-legal-content.md), [ADR-0033](0033-runtime-module-resource-gating.md)

## Decision

The optional `analytics` capability stores only a SHA-256 digest of an opaque
consent handle and
purpose-versioned decision, then aggregates validated events by UTC day,
event type, country and purpose version. It never stores IP addresses,
identities, URLs, fingerprints or per-visitor event rows. `RecordEvent` fails
closed unless the handle has a current granted decision for the same purpose
version. `PurgeBefore` is explicit and operator-selected because legal
retention differs by deployment.

The migration is module-scoped and therefore absent from Compact deployments
without analytics. SQLite and PostgreSQL use equivalent keys and constraints;
the provider-specific SQL remains inside the optional adapter.

## Evidence and limits

`internal/analytics/sqlstore/store_test.go` proves consent gating, aggregation
and revocation behavior on SQLite. Legal text, DPO approval, retention period,
export/erasure policy and production privacy review remain required gates.
