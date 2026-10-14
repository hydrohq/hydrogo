# ADR-0296: Tenant-scoped content view totals in Admin Studio

- **Status:** Accepted
- **Date:** 2026-09-29
- **Related:** [ADR-0217](0217-consent-gated-content-view-analytics.md), [ADR-0290](0290-final-design-form-and-discovery-capabilities.md)

## Decision

Admin page and news lists may show and sort by lifetime content view totals only
when the optional analytics capability is enabled. Totals are derived from the
existing consent-gated daily aggregate; no visitor identifiers or raw request
data are read or exposed. Each bounded query is constrained by tenant, content
type, and at most 100 listed content IDs.

The analytics migration adds a covering index for that lookup. SQLite,
PostgreSQL, and MySQL share the same indexed column order and retain their
existing aggregate contracts. Inactive analytics adds neither a query nor a
list column.

## Limits

Counts represent only views recorded after analytics consent. They are a
privacy-preserving editorial signal, not a billing or unique-visitor metric.
Sorting applies to the bounded Admin Studio page currently displayed.

## Verification

Tests cover tenant and content-type isolation, bounded ID lists, typed safe sort
links, and end-to-end Admin Studio rendering. The migration is checked against
SQLite/PostgreSQL/MySQL metadata; full verification runs `templ generate` and
`go test -count=1 ./...`.
