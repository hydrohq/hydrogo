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
- Onboarding progress and dismissal are scoped by role. The optional guide
  destination is HTTPS-only configuration and opens from an accessible Studio
  guide tab.
- Content history, translation, imports, and bulk operations stay module-owned.
  Imports are bounded to 5 MiB, 5,000 data rows, 100 columns, and 16 KiB per
  cell. XLSX support reads the first worksheet using the Go standard library.
- Revision history stores the actor ID privately and resolves an email only
  for authorized Admin Studio views. Visual previews are sandboxed without
  script permission.
- The notification badge reads an authenticated unread count. Notification
  records and mark-read/dismiss operations are user and tenant scoped.
- The optional support card is static configuration and has no runtime worker.
- The page module owns the notification inbox migration. Inbox queries and
  mutations always scope to both the authenticated user and, when present, the
  tenant. Notification targets must be local paths.
- Weekly summaries default to disabled and save an explicit per-user opt-in
  and IANA timezone. The runner activates only when Pages, Forms, and SMTP
  are enabled. It uses the existing mail spool and atomically claims each
  user/reporting week with the enqueue operation.
- The report covers the preceding local calendar week: new form submissions,
  open leads, and (when Analytics is active) the five leading referrers.

## Compatibility and operations

Inbox, digest preference, and revision attribution migrations are additive and
transactional for SQLite and PostgreSQL. Old application versions ignore the
new tables/column during blue/green rollout; the actor column has an empty
default for old writers.
The inbox table grows with notifications; a retention policy is required before
high-volume event producers are enabled. Digest scheduling is a minute-level
hook on the existing mail-spool worker and creates no additional worker.

## Verification

SQLite behavior is covered by inbox, digest, and revision attribution contract tests. PostgreSQL SQL is kept
dialect-neutral apart from placeholders and timestamp types; the certified
PostgreSQL contract runs when `HYDRO_TEST_POSTGRES_URL` is configured.

## Rollback

Use a forward migration or restore. Dropping the inbox table discards user
notification history and is not an automatic rollback.
