# ADR-0044: Deterministic capability kit sets

## Status

Accepted

## Decision

Client Edition and Admin Studio composition may validate kit sets through
`ValidateKits`. Unknown or duplicate capability names fail closed, avoiding
ambiguous activation and making generated metadata deterministic.

## Compatibility and capacity

Validation is additive and migration-free for SQLite/PostgreSQL. It is an
in-memory bounded check and adds no routes, workers, assets, or cache state.
