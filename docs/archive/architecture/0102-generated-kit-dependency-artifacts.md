# ADR-0102: Generated kit artifacts include resolved dependencies

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Kit dependency resolution existed as a deterministic Admin Studio contract,
but generated runtime wiring, kit metadata, and admin manifests still listed
only the kits explicitly written in a collection spec. A generated project
could therefore omit a required companion capability from its review surface.

## Decision

The generator resolves the allowlisted kit dependency graph before rendering
admin metadata, `kits.ts`, and `RUNTIME_WIRING.md`. The output is sorted and
contains each dependency exactly once. Resolution remains metadata-only:
implementations, routes, migrations, workers, and resource budgets are still
owned by the Client Edition/module composition root.

## Consequences

Generated artifacts now describe the complete capability set that must be
bound, while inactive kits still have zero runtime footprint. Adding a new
dependency requires the existing explicit registry and contract-test change.

## Verification

`go test ./internal/generator ./internal/adminstudio -count=1` verifies
deterministic ordering, Commerce → Forms expansion, and generated artifact
coverage.
