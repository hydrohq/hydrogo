# ADR-0024: Generated artifact compatibility checks

- Status: Accepted
- Date: 2026-09-15

## Context

HydroGo generates reviewable source, migrations and client artifacts from a
versioned collection spec. A later upgrade must detect stale or partially
deleted output before it is deployed, without rewriting user code.

## Decision

The generator exposes a read-only compatibility check. `hydro validate
generator --spec SPEC --root OUTPUT` validates the spec hash in
`generated/generator.lock` and the complete required artifact set. It fails
closed on missing, symlinked or non-regular artifacts. The check is safe to run
in CI and is independent of database/cache providers.

## Consequences

Generated output remains deterministic and migrations remain immutable, while
upgrade workflows receive an actionable preflight gate. This does not infer
business compatibility or replace reviewed migration/ADR decisions; those
remain explicit module responsibilities.
