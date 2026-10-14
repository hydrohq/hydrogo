# ADR-0041: Capability kits in E2E diagnostics

## Status

Accepted

## Decision

The protected Admin Studio E2E manifest includes the selected capability kits
for each collection. Browser tooling can show or exercise the right surfaces
without exposing provider credentials or implementation details.

## Compatibility and capacity

The JSON field is additive and migration-free for SQLite/PostgreSQL. It is
metadata only and does not activate routes, workers, assets, or cache entries.
