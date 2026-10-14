# ADR-0094: Typed frontend query adapter contract

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Generated TypeScript clients already exposed list requests, but React output
only returned an untyped placeholder object. This forced each Client Edition
to rewrite query keys and fetcher wiring, slowing standard CRUD prototypes.

## Decision

The generator emits a framework-neutral typed query-options factory and stable
query-key helper in `react.ts`. It may be consumed by React Query, SWR, or a
custom hook. React and any query library remain peer dependencies; HydroGo Core
does not bundle a browser runtime. Query keys are deterministic and callers
remain responsible for authentication, cache invalidation, and policy.

## Consequences

- New generated resources have typed query wiring immediately available.
- No runtime or dependency footprint is added to Compact deployments.
- The adapter does not guess business rules or silently perform mutations.
- Full browser hook acceptance remains a Client Edition/product test.

## Verification

`go test ./internal/generator` verifies the generated adapter contains the
typed options and key contract, and the generated fixture still compiles.
