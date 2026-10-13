# HydroGo Incident Response Template

This template is filled during operational incidents; sensitive values, tokens, and personal data must never be written here.

## Incident Summary

- Start Time (UTC):
- Reporter / Incident Commander:
- Affected Edition / Profile:
- User Impact:
- Initial Request ID / Alert Name:

## Safe Immediate Steps

1. Verify blast radius using `/healthz`, `/readyz`, metrics, and redacted log correlation.
2. Halt pending deployments, migrations, and write operations; do not delete data without preserving evidence.
3. Record state of sessions, media, worker queues, and backups separately.

## Degraded Mode Decisions

- Activation condition and approver:
- Disabled capability / module:
- User-facing message:
- Recovery / restoration criteria:

## Rollback / Restore

- Target binary / release manifest:
- Migration compatibility verification:
- Backup manifest verification outcome:
- Session revocation / write fence / egress quarantine status:
- Rollback approver:

## Incident Closure

- End Time (UTC):
- Verified SLO / SLI impact:
- Permanent remediation task:
- Associated ADR / validation report:
- Sensitive data leak review outcome:
