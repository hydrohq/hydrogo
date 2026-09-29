# ADR-0320: Incident Safe Mode Runtime

Status: Accepted

## Context

Operators need a reversible startup control that lets administrators recover
the site while external delivery and optional background work are paused.
Safe Mode must preserve normal password authentication, local content editing,
session cleanup, and security controls.

## Decision

Safe Mode is enabled by YAML `safe_mode`, `HYDRO_SAFE_MODE`, or `--safe-mode`.
Precedence follows the runtime configuration contract: defaults, YAML,
environment, then explicit command-line flags. The Admin Studio shell shows a
high-visibility warning while enabled.

Safe Mode does not configure social OAuth, push providers, SMTP/MX mail
delivery, or scheduled S3 backups. It does not start mail, news publication,
webhook outbox, or monthly outbound link audit workers and does not dispatch
those jobs in CGI lazy cron. Durable queued records remain stored for later
delivery. Password login, MFA, local session cleanup, IP blocklist refresh,
content routes, and recovery editing remain active. No custom script injector
exists in this runtime yet; when one is added it must consult the same flag.

## Compatibility, capacity, and recovery

The change adds no schema or migration and uses the existing configuration
loader, so SQLite, PostgreSQL, and MySQL schemas are unaffected. It adds no
worker, route, or required service. Existing queue rows are retained and can
resume after restarting without Safe Mode. Blue/Green instances must use the
same mode intentionally: a normal-mode instance can still deliver queued
external work. The setting is process-wide, not dynamically toggled.

Operators can disable Safe Mode by removing the flag or setting the YAML/env
value false and restarting. The Admin Studio warning identifies the active
state. Recovery that depends on outbound email is unavailable while delivery
is paused; password and MFA login remain available.

## Verification

Configuration precedence and invalid values are unit-tested, the shared Admin
Studio warning is tested, and the full Go package suite plus migration guard
are release checks. No database migration is required.
