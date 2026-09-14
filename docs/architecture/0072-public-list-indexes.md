# ADR-0072: Index bounded public and admin list reads

## Status

Accepted

## Decision

Add separate, module-scoped dialect-specific partial and ordering indexes for
page and news list queries. Public indexes include locale/status and published
state; admin indexes support the bounded `updated_at, id` ordering. A local
SQLite planner fixture guards against regressions; PostgreSQL planner and
production sizing remain release gates. Keeping migrations module-scoped means
an inactive module does not require the other module's tables.

## Consequences

List endpoints avoid predictable full scans as content grows, at the cost of
four maintained secondary indexes. Migration metadata requires operators to
estimate disk and lock impact before production rollout.
