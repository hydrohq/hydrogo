# ADR-0050: Validate module dependency graphs at the SDK boundary

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0049](0049-versioned-module-route-contract.md)

## Decision

`extension/v1.ValidateDependencyGraph` validates normalized module dependency
maps before runtime resolution. It rejects empty names, unknown dependencies,
and cycles, while treating duplicate dependency entries as idempotent.
Runtime registration and lifecycle remain Core responsibilities.
