# ADR-0125: Versioned analytics consent contract

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0022](0022-consent-based-product-analytics-and-legal-content.md), [ADR-0013](0013-identity-and-security-boundaries.md)

## Decision

Client Editions use `extension/v1` for explicit purpose-versioned analytics
consent and a minimized event vocabulary. The contract has no visitor identity,
IP, full URL, account ID, fingerprint or advertising ID fields. A durable
analytics module must reject events without current consent, aggregate by day,
apply retention and provide revocation semantics; this SDK contract does not
create tables, routes or cookies.

## Evidence and consequences

`extension/v1/analytics_test.go` covers consent metadata, revocation and
aggregate-safe event validation. Durable persistence, legal text and legal
review remain open production/product gates.
