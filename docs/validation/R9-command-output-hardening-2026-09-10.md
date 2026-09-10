# R9 command output hardening — 2026-09-10

**Status:** Implemented and locally verified.

## Change

PostgreSQL backup and restore invoke external `pg_dump` and `pg_restore`
processes. Their combined stdout/stderr output is now capped at 64 KiB. The
process still receives its full exit status, while diagnostic text is marked as
truncated when a broken database, provider, or backup emits excessive output.

This prevents an untrusted or failing external command from causing an
unbounded memory allocation in the HydroGo process.

## Verification

The following checks passed on Windows with Go 1.27.0:

```text
go test ./internal/backup -count=1 -timeout=5m
go vet ./internal/backup
git diff --check
```

Regression coverage verifies that output larger than the cap is retained only
up to the configured bound and reports truncation.

## Scope

This hardening is limited to backup command diagnostics. It does not change
backup formats, database contracts, restore controls, or deployment topology.
