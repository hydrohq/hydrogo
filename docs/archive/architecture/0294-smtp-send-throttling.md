# ADR-0294: Bounded SMTP send rate

- Status: Accepted
- Date: 2026-09-29

## Context

Shared SMTP services may defer or suspend a sender that emits a burst of
messages. The durable mail spool retries failed sends, but the SMTP adapter did
not pace its own work.

## Decision

Each configured SMTP sender applies two in-process token buckets before
opening a network connection. The defaults allow a burst of five messages and
refill at five per second, plus a one-token bucket refilling at 100 per hour.
`mail.max_per_second` and `mail.max_per_hour` (or the corresponding
`HYDRO_SMTP_MAX_PER_SECOND` and `HYDRO_SMTP_MAX_PER_HOUR` environment variables)
override these limits. Waiting observes context cancellation and is excluded
from the SMTP network timeout; cancellation/retry remains governed by the
existing spool worker.

The limiter is constructed only when SMTP is enabled, uses bounded constant
memory, and adds no route, migration, cache key, or background goroutine.

## Consequences

SQLite and PostgreSQL behavior is identical and no schema change is needed.
The configured limit applies per process: deployments with multiple active
instances should divide the provider's total quota across instances. Blue/Green
rollouts temporarily run both process-local quotas, so the per-instance setting
must account for overlap. Rollback is a configuration/code rollback; no
persistent state needs recovery. Token reservation and cancellation behavior
are covered by unit tests.
