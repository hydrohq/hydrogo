# ADR-0067: Require explicit confirmation for demo seeding

## Status

Accepted

## Decision

The production image guidance copies only the runtime binary, configuration,
lock/license metadata, themes, and migrations. Demo records are not copied.
The `seed-demo` command additionally requires `--confirm-demo` so a deployment
cannot seed demo accounts by accident.

## Consequences

Local E2E and disposable development databases must opt in explicitly. Demo
seed code remains available for developer ergonomics, but no production data
or credentials are treated as fixtures.
