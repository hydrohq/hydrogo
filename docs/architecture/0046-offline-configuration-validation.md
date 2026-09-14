# ADR-0046: Offline configuration validation command

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0007](0007-capacity-profiles.md), [ADR-0033](0033-runtime-module-resource-gating.md)

## Context

Developers and CI need a fast way to catch invalid configuration before a
database, Redis, media encoder, or other runtime dependency is available.
`hydro doctor` intentionally performs dependency and schema checks, so it is
not suitable for an offline edit loop.

## Decision

Add `hydro validate [--json]` to load and validate configuration, capacity
profile, and module normalization without opening external services or printing
secret values. Module names are returned in deterministic sorted order. The
existing `doctor` command remains the runtime/dependency diagnostic.

## Consequences

- CI and pre-commit workflows get deterministic, dependency-free validation.
- A successful validation does not claim database compatibility or production
  readiness; those remain `doctor` and external gate responsibilities.
- JSON output can be consumed by editors and automation without exposing DSNs or
  secrets.
