# ADR-0272: Admin Studio business UX and bounded inbox

## Status

Accepted

## Context

Client Editions need a business-owner view without exposing developer, database,
or internal identifier details. Module screens remain responsible for their
own authorization and mutations. Optional agency support content must not add
services or background work when disabled.

## Decision

- The editor shell uses an allowlisted content navigation set. Admin-only
  screens remain available to administrators through the existing role policy.
- Shared quick actions are rendered by typed Templ components and use the
  existing CSRF-protected credential endpoint.
- Content history, translation, imports, and bulk operations stay module-owned.
  Imports are bounded to 5 MiB, 5,000 data rows, 100 columns, and 16 KiB per
  cell. XLSX support reads the first worksheet using the Go standard library.
- The optional support card is static configuration and has no runtime worker.
- The page module owns the notification inbox migration. Inbox queries and
  mutations always scope to both the authenticated user and, when present, the
  tenant. Notification targets must be local paths.
- Weekly summary delivery is not part of this decision until an opt-in,
  timezone, aggregation, retry, and retention contract is accepted.

## Compatibility and operations

The inbox migration is additive and transactional for SQLite and PostgreSQL.
Old application versions can ignore the new table during blue/green rollout.
The table grows with notifications; a retention policy is required before
high-volume event producers are enabled. No worker is added to Compact.

## Verification

SQLite behavior is covered by the inbox contract test. PostgreSQL SQL is kept
dialect-neutral apart from placeholders and timestamp types; the certified
PostgreSQL contract runs when `HYDRO_TEST_POSTGRES_URL` is configured.

## Rollback

Use a forward migration or restore. Dropping the inbox table discards user
notification history and is not an automatic rollback.
