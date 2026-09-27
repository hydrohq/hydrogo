# ADR-0267: Form leads and durable mail spool

- **Status:** Accepted
- **Date:** 2026-09-27
- **Related:** [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0024](0024-optional-redis-rate-limiting-and-queue-boundary.md), [ADR-0121](0121-versioned-forms-contract.md), [ADR-0134](0134-durable-form-registry.md), [ADR-0176](0176-tenant-scoped-form-registry.md), [ADR-0183](0183-forms-media-rls-call-sites.md), [ADR-0190](0190-capacity-profiles-concurrency-budgets.md), [ADR-0270](0270-direct-mx-mail-fallback.md)

## Context

The optional Forms capability already owns validated schemas and the Site
module owns durable submissions. A form submission must remain durable when
SMTP is unavailable, and its lead-management fields and private staff notes
must remain tenant-scoped. Sending email inside a database transaction is not
allowed. The generic SQL jobs queue remains the contract for scheduled jobs;
mail needs a small operator-facing delivery record with recipient, retry time,
and a manual retry action.

## Decision

When Forms is active, a form submission, audit record, and its bounded mail
spool entries are written in one SQLite/PostgreSQL transaction. SMTP delivery
is at-least-once and happens after commit. A lost SMTP response can still cause
a duplicate message; no exactly-once promise is made. The mail spool stores a
bounded text message and uses a six-attempt schedule: immediate, then 5 minutes,
15 minutes, 1 hour, 6 hours, and 24 hours. Permanent transport errors and
exhausted retries become `failed_permanent`. A manual retry starts a new retry
cycle and retains the previous bounded diagnostic until a new attempt finishes.

The spool claims due rows with expiring leases and fencing tokens. SQLite and
PostgreSQL stores use parameterized SQL and tenant scope; PostgreSQL enables
RLS for the new notes and spool tables. Active spool rows are capped by the
existing capacity profile queue budget. Delivered records are eligible for
bounded cleanup after 30 days; permanently failed records remain visible to
operators until explicitly retried or removed under the configured retention
policy. Worker concurrency is at most two and cannot exceed the selected
profile's worker budget. It starts only when Forms and a mail sender are both
enabled. The sender may be the explicitly configured direct MX path defined by
ADR-0270.
CGI does not start a resident worker; existing CGI lazy cron may process a
bounded due item within its request budget.

Form inbox records gain separate lead stage and assignee fields; existing inbox
status keeps its current meaning. Staff notes are append-only records with
author and timestamp. Public APIs never project lead stages, assignees, or
internal notes. Honeypot submissions are silently acknowledged and never
persisted or dispatched. Turnstile verification is optional, calls only the
fixed Cloudflare endpoint with a short timeout and bounded response, and is
disabled unless explicitly configured. Idempotency keys are required for
public submissions and scoped by tenant, form, and token.

## Data and deployment compatibility

The expand migration adds nullable-compatible/defaulted lead fields, fixes the
idempotency uniqueness scope, and adds tenant-scoped notes and `mail_spool`
tables. No existing submission or form JSON is rewritten. Both SQL dialects
have explicit metadata, checksums, smoke tests, and rollback notes. Old
binaries can read the existing inbox columns but do not understand the new
lead workflow or spool; do not enable new writes until old binaries are
drained. Returning to an older binary after lead or spool writes requires
keeping the new tables and columns in place and disabling the new UI/worker;
dropping them would lose queued mail and lead metadata.

## Consequences

- SMTP downtime does not discard an already accepted submission or queued
  notification.
- Queue saturation rejects the whole transaction with a retryable service
  response; the system never reports a successful submission whose requested
  notifications were silently discarded.
- Spool payloads contain personal data and use the same access controls and
  backup/retention protections as the form inbox.
- Retry delivery is at-least-once. Operators may see a duplicate after an
  ambiguous provider timeout.
- Inactive Forms creates no spool worker, routes, assets, or migration.
