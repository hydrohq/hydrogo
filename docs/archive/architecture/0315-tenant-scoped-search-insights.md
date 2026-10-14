# ADR-0315: Tenant-scoped site search insights

## Status

Accepted

## Decision

When both Search and Analytics are enabled, submitted public `GET /search`
queries may contribute to daily per-tenant, per-locale aggregates.
Autocomplete requests are excluded. The aggregate stores a normalized term,
locale, search count, and zero-result count; it stores no account, visitor, IP,
or device identifiers.
Terms resembling email addresses or phone numbers are discarded before they
reach the queue. The UI reports 7- and 30-day totals and requires an active
admin session and tenant scope where the deployment requires one.

Recording runs through a bounded in-process queue (256 events, one worker).
Full queues drop analytics events rather than delaying search responses. The
queue drains on orderly shutdown. Analytics retention removes search rows at
the same operator-selected cutoff as other daily aggregates. No cache or
independent worker is added. The migration and admin report are gated by the
optional Analytics module; the public search route is gated by Search.

SQLite, PostgreSQL, and MySQL use the same `(day, tenant_id, term)` aggregate
key. PostgreSQL row-level security follows the tenant setting used by the
application database contract. The additive migration is ignored by older
binaries; rollback drops the table and its aggregate history. The index orders
by day and tenant to bound report and retention access. Space growth is bounded
to one row per active tenant, day, and distinct normalized submitted term.

## Verification

Adapter contract tests cover normalization, locale aggregation, zero-result counts,
tenant separation, and retention on SQLite and MySQL, with PostgreSQL enabled
by its integration DSN. Queue tests verify asynchronous persistence and term
filters. HTTP tests cover public search routing and admin authorization.
