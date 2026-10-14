# ADR-0309: Site not-found tracking and published slug redirects

## Status

Accepted

## Context

Site operators need to find broken inbound links and preserve links when a
published page changes its URL. Tracking must not retain request query strings,
referrers, or client addresses, and it must not allow unbounded path growth.

## Decision

The optional Site module records canonical missing paths in a tenant-scoped
aggregate table. It stores hit count and first/last timestamps, caps distinct
paths at 5,000 per tenant, and exposes a bounded report and explicit conversion
to an exact 301 redirect. Query strings, referrers, and client addresses are
discarded. Private route families are excluded from collection.

When a page is published with a changed default-locale slug, the page transport
records a permanent redirect from the prior published slug. Draft edits do not
create redirects. Redirect replacement validates exact, wildcard, and regular
expression rules against representative paths and rejects detected loops.

## Compatibility and operations

SQLite, PostgreSQL, and MySQL use the same tenant/path-hash uniqueness contract.
PostgreSQL applies tenant RLS. No worker or additional service is introduced;
tracking runs in the existing request path and only when Site is enabled. The
report query is limited to 500 rows. Operators can remove individual entries
after conversion.

Rollback drops `site_not_found_hits`; its aggregate diagnostics are disposable.
The migration manifest records dialect checksums. Existing redirects and page
content remain readable during rollout; slug redirect persistence occurs after
the page publish succeeds.

## Validation

Repository, transport, redirect-loop, tenant isolation, and publish-hook tests
cover the behavior. Migration contract checks validate the dialect migration.
