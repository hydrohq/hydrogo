# ADR-0307: Monthly Executive Digest

## Status

Accepted.

## Context

Administrators already opt in to a weekly site digest. The reporting request adds a monthly cadence and traffic, top-page, inquiry, and optional member counts. Delivery must remain bounded and safe in Compact deployments.

## Decision

Extend the existing preference with a weekly/monthly frequency. Existing preferences migrate with `weekly`, preserving their cadence. A new administrator preference defaults to monthly in the settings UI but remains disabled until explicitly enabled. The existing scheduler polls enabled preferences and selects the previous complete local calendar period after 09:00 on Monday or the first day of a month. A period claim is written in the same transaction as the durable mail enqueue, preventing duplicate delivery across scheduler ticks and processes.

Monthly analytics use the tenant-scoped daily aggregates already maintained by the analytics capability. Member counts are queried only when that module is enabled; inquiries are read from the existing tenant-scoped form inbox. The digest reports totals and at most three popular pages, and does not add raw visitor identifiers or a new worker. Missing analytics or member capabilities contribute no data and do not start services.

## Database and rollout

Migration `202609290006_admin_digest_frequency` adds a small frequency column. SQLite and PostgreSQL constrain values to `weekly` or `monthly`; MySQL uses a bounded varchar and application validation. All supported dialects default old rows to weekly. This is an additive expansion migration, readable and writable at schema epoch 1. Rollback is forward-only; restoring the prior binary requires reverting the application first and retaining the new column, which old code ignores. The write path uses each dialect's native upsert syntax.

## Capacity, security, and operations

Preference scans remain keyset-paginated and capped at 500 rows. Metrics are aggregated in SQL over existing indexed daily rows. Tenant scope is applied to forms, analytics, preferences, and member rows. No new cache, worker, route, or module activation is introduced. Existing durable mail backpressure and delivery controls continue to apply. SQLite, PostgreSQL, and MySQL migration variants preserve the shared schema contract.
