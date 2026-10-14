# ADR-0124: Versioned capability kit contract

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0036](0036-optional-capability-kit-registry.md), [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md)

## Decision

`extension/v1` exposes optional capability kit names, dependency resolution and
deterministic descriptors. Resolving a kit set is metadata-only: it must not
implicitly register routes, migrations, assets or workers. Client Editions bind
implementations explicitly, preserving zero-footprint behavior for omitted kits.

## Evidence and consequences

`extension/v1/kits_test.go` covers dependency expansion, deterministic ordering,
unknown-name and duplicate rejection. Provider implementations and capacity
impact remain Edition/module responsibilities.
