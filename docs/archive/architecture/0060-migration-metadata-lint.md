# ADR-0060: Lint migration safety metadata before release

## Status

Accepted

## Decision

`tools/migrationguard` requires every migration to declare both dialect
transactionality values, checksums, lock-timeout evidence, disk estimate,
compatibility notes, and rollback strategy. Missing values fail closed.

## Consequences

Migrations must be reviewed before passing local verification. Production lock
measurements and rollback drills remain environment-specific gates.
