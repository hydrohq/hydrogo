# ADR-0326: Privacy-Safe Form Conversion Analytics

Status: Accepted

## Context

Form owners need a useful view-to-submission conversion rate and a short trend
without retaining raw visitor identifiers. The existing optional no-cookie
analytics mode already defines operator consent, daily HMAC pseudonyms, and
retention behavior.

## Decision

Count form views only when Analytics no-cookie tracking is enabled and its
operator-provided hash key is configured. The typed public form includes a
same-origin, CSRF-protected view request. A bounded rate limiter and
`analytics_form_visitors` unique key deduplicate by day, tenant, form, and
daily-pseudonymous visitor. The raw address and user agent are never stored.
The daily aggregate contains only form view and accepted-submission counts.
Replayed idempotent submissions do not increment submissions.

Admin Studio shows 30-day conversion percentages and per-day submission bars.
Reads and writes are tenant-scoped; PostgreSQL uses row-level security. The
existing analytics retention worker removes both deduplication and aggregate
rows according to the configured retention dates.

## Consequences

The additive migration creates daily aggregate and deduplication tables when
Analytics is enabled. Existing binaries ignore both tables. The view endpoint
is only registered when Forms is active, and no new worker or service is
required. If no-cookie analytics is disabled, the report is omitted rather
than recording unidentified or raw visitor data.

## Database and Capacity

SQLite, PostgreSQL, and MySQL have certified insert, deduplication, tenant
separation, and report-query contracts. Each accepted visitor adds one bounded
daily deduplication row per form; both tables are covered by existing analytics
retention. Writes use a short transaction, and report reads are limited to 30
days and 1,000 rows. SQLite uses its existing transaction and WAL policy;
PostgreSQL RLS and tenant-aware adapter context preserve tenant boundaries.

## Recovery and Rollback

Rollback drops `analytics_form_visitors` and `analytics_form_daily`, losing
form analytics history but leaving forms and submissions intact. The additive
schema is safe for blue/green rollout because old binaries do not read or
write the new tables.
