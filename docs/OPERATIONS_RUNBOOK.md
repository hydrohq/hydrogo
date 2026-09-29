# HydroGo Operations Runbook

This document compiles operational procedures for the Compact pilot. Restore and production capacity claims are not certified without empirical environment evidence recorded in the corresponding validation report.

## Deployment and Rollback

1. Copy the new binary, config, and migration manifest to the staging directory; execute `hydro migrate`.
2. In production, configure HTTPS `HYDRO_PUBLIC_ORIGIN` alongside `HYDRO_SECURE_COOKIES=true` and `HYDRO_ENABLE_HSTS=true`; if any of these three is missing or contradictory, configuration fails on startup. Verify that HTTPS termination is active. Capture `hydro version --json` output into deployment records to persist the binary commit hash and embedded schema migration level.
3. Verify the selected capacity profile, DB connectivity/checksums, backup tooling, and media directory writability using `hydro doctor`; start `hydro serve` and wait for `/readyz`.
4. The `/metrics` endpoint is protected by authenticated operations access and never cached; HTTP request totals, in-flight gauges, bounded status classes, readiness state/failure counters, worker outcomes, and session cleanup metrics are scraped by alerting/capacity systems. `hydro_readiness_state 0` indicates an unready instance; an increase in `hydro_readiness_failures_total` reflects DB or migration readiness failure history.
5. In every release validation, run `go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...`; if reachable vulnerability findings are not clean, halt the release and evaluate advisories. Unreachable module-level findings are documented in [dependency vulnerability review](security/DEPENDENCY_VULNERABILITY_REVIEW.md).
6. `hydro serve` writes structured JSON HTTP logs to stderr. Logs contain request ID, HTTP method, path, status, and duration; query strings, request bodies, cookies, and response bodies are excluded.
7. During Blue/Green deployment, execute smoke queries against public routes, private media, login, sitemap, and form submissions on the green slot before routing live traffic.
8. Stop the old slot such that it cannot accept new writes; background worker leases are allowed to drain/expire naturally.
9. In failure scenarios, binary rollback is executed only if compatibility with the old schema and content/event/job formats is confirmed. Restoring from a backup is a distinct recovery procedure.

## Compact Snapshot and Restore

Writes and worker external egress must be paused in a controlled manner. Use `hydro backup sqlite` for SQLite and `hydro backup postgres` for PostgreSQL; SQLite uses `VACUUM INTO`, PostgreSQL uses `pg_dump` custom format. In both workflows, media files are copied alongside a checksum manifest. The PostgreSQL password is provided via temporary `PGPASSFILE` and never passed in command-line arguments.

The restore target must be an empty staging directory. No files are activated before passing `Verify`. `RestoreToStaging` creates the activation marker only when all three operational controls are proven:

- Old session records are revoked.
- Workers and external egress are quarantined.
- Old deployment writes are blocked (write fence).

Without the marker, staging cannot be activated for public traffic, workers, or export operations. Use `hydro restore stage` for SQLite, and `hydro restore postgres` targeting an empty target DB/instance for PostgreSQL; renaming directory paths is not a PostgreSQL restore. `hydro restore verify` executes checksum verification independently prior to activation.

## Password Recovery and Invitations

Reset and invite tokens are stored exclusively as hashes. Tokens are never written to logs or audit records once passed to the email/notifier adapter. Unknown password reset requests receive an identical `202 Accepted` response. Token consumption and user creation during invite acceptance execute in a single database transaction.

In production, SMTP notifier settings (`HYDRO_SMTP_HOST`, `HYDRO_SMTP_PORT`, `HYDRO_SMTP_FROM`, `HYDRO_SMTP_USERNAME`, `HYDRO_SMTP_PASSWORD`, and `HYDRO_SMTP_STARTTLS`) are validated together. A sender-only `HYDRO_SMTP_FROM` opts into direct MX delivery for staging and low-volume use; it requires outbound port 25 and receiver acceptance. Forms email remains in the durable spool for retry when delivery fails. Use an authenticated submission provider for production; direct MX does not configure SPF or DKIM (see ADR-0270).

## Forms and Mail Spool

The optional Forms module accepts public form requests only when enabled. Configure
`HYDRO_FORMS_TURNSTILE_SITE_KEY` and `HYDRO_FORMS_TURNSTILE_SECRET_KEY` together
to enable Turnstile; leaving both unset disables the challenge. Keep the secret
key in the deployment secret store. Form alerts and autoresponders are committed
to the tenant-scoped SQLite/PostgreSQL mail spool with the submission, then sent
by the worker when an SMTP or explicitly enabled direct MX sender is configured.
Monitor `/admin/site/mail` for permanent delivery failures and use its retry action after fixing provider configuration.
Delivered spool records are eligible for cleanup after 30 days; failed records
remain visible for operator action. Queue capacity follows the active capacity
profile, and a full queue rejects the submission without losing its payload.

## Disk Space Exhaustion

New uploads and snapshot operations are halted with a safe error; existing ready media and database files are never deleted. Temporary/pending media, failed snapshots, and expired inbox records are cleaned up first according to retention policies. Migrations, restore activations, and GC are blocked until the filesystem is confirmed writable again.

For snapshot retention, first inspect candidates with `hydro backup prune --root SNAPSHOTS --keep 3 --dry-run`; delete candidates only after `--confirm`. Pinned snapshot names are specified via `--pinned` and are never deleted under any circumstance. Missing or corrupt snapshot manifests fail closed and halt pruning.

## Alert Evaluation

`internal/ops.Evaluate` converts last backup age, available disk space, migration/readiness failures, SMTP delivery failures, and dead-letter queue metrics into deterministic alarm codes. `backup_stale`, `disk_low`, `migration_failed`, and `readiness_failed` are critical; `smtp_delivery` and `jobs_dead_lettered` are warning level. The evaluator does not replace an alerting notification system; production notifiers and external key recovery drills must be verified separately.

Admin Studio's `/admin/backup` download requires an operator-managed `HYDRO_BACKUP_SIGNING_SEED` (64 hexadecimal characters) supplied through the deployment secret store. The page shows the matching public verification key. Store that public key with each downloaded archive; do not rotate away old public keys until their archives expire. Verify and unpack a downloaded archive before staging it:

```sh
hydro restore unpack --archive hydrogo-backup.tar.gz --destination /secure/restore/snapshot --public-key PUBLIC_KEY_HEX
hydro restore verify --snapshot /secure/restore/snapshot
```

News worker concurrency is derived from the active capacity profile: Compact uses 1 worker, Standard uses 4, Distributed uses 16 bounded worker IDs. Profile concurrency should not be overridden without measuring lease durations and queue backpressure.

Dead-letter jobs should first be inspected with `hydro jobs dead-letter list`. Job payloads are omitted from CLI outputs. Operators re-queue jobs only with explicit confirmation after verified fixes: `hydro jobs dead-letter requeue --id ID --confirm`; attempt counters and error diagnostics are preserved.

Rate-limit responses return `429` with a `Retry-After` header; clients and reverse proxies must respect this backoff. Login/MFA and form limiters reject empty keys; distributed limiters do not forward empty keys to backends.

Unneeded dead-letter records are purged only with an explicit retention age and batch limit: `hydro jobs dead-letter purge --older-than 720h --limit 100 --confirm`. Active and claimable jobs are unaffected.

## Current Limits and Budgets

- Login/MFA: process-local limiter, 10,000 keys, 5 attempts per key / 1 minute.
- Contact form: 10,000 keys, 5 submissions per form+IP / 1 minute; body max 64 KiB.
- Media upload: 10 MiB limit.
- Public page cache: 256 items, 30-second TTL.
- Sessions: 7-day absolute TTL, 24-hour idle TTL.
- Expired sessions are cleaned up at startup and every 5 minutes by a background worker; `hydro_session_cleanup_runs_total`, `hydro_session_cleanup_rows_total`, and `hydro_session_cleanup_errors_total` metrics are tracked.

These defaults must be measured under pilot traffic and updated in the R3 capacity report; defaults do not constitute an empirical hardware or RPO/RTO guarantee.

## Personal Data Inbox Requests

Form inbox access, export, and erasure operations are governed by [PERSONAL_DATA_POLICY.md](PERSONAL_DATA_POLICY.md). The operator verifies admin privileges and request identity before opening the audit record; exports are placed in encrypted temporary storage and deleted after delivery. Erasure operations are not marked complete until replayed against the external erasure checkpoint.
