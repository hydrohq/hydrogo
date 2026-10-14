# ADR-0314: Aggregate content feedback

## Status

Accepted

## Decision

The optional Analytics module may record explicit helpful/not-helpful reactions
for news and FAQ content. It stores only daily tenant-scoped aggregate counts;
visitor identifiers, IP addresses, and device data are not persisted. The
public endpoint is CSRF protected by the shared middleware and rate limited.
Tenant scope is mandatory in multi-tenant deployments. The public browser keeps
a local duplicate-vote guard, which is a UX aid rather than a trust boundary;
repeated submissions from other clients count as separate votes within the
configured rate limit.

The new table is additive and enabled only with Analytics. SQLite, PostgreSQL,
and MySQL share the same logical key and aggregate semantics. Existing binaries
ignore the table. The explicit analytics retention operation removes feedback
rows at the same cutoff as existing daily analytics. A day index keeps that
deletion bounded. The feature adds no worker or cache.

## Verification

Certified database contract tests cover aggregation, tenant isolation, invalid
reactions, and content count queries. Transport tests cover method/input
validation, CSRF through the shared middleware, rate limiting, and rendered
responses. Admin Studio exposes aggregate news article and page FAQ counts.
