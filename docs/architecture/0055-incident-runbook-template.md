# ADR-0055: Structured incident and rollback runbook template

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0011](0011-consistency-upgrade-and-recovery.md), [ADR-0016](0016-backup-restore-and-data-lifecycle.md)

Operational incidents use a redacted, evidence-first template covering health,
deploy/migration freeze, degraded mode, backup verification, session revocation,
write fences, and rollback approval. The template contains no target SLO values
and does not authorize destructive actions by itself.
