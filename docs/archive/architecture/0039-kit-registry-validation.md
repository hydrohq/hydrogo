# ADR-0039: Fail-closed kit metadata validation

## Status

Accepted

## Decision

The Admin Studio registry validates capability kit names against the shared
vocabulary. Unknown values are rejected before rendering or route composition,
keeping provider internals out of Core metadata.

## Compatibility and capacity

Validation is additive and migration-free for SQLite/PostgreSQL. It is an
in-memory lookup and does not activate services or change Compact profile
resource budgets.
