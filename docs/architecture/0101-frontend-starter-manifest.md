# ADR-0101: Frontend starter manifest

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Generated Client Editions have themes and API contracts, but no single stable
artifact tells a frontend team which preset, presentation mode, features, and
design-system adapters are available.

## Decision

`hydro project new` emits `frontend/starter.json` and a short frontend README.
The manifest is deterministic, credential-free, and lists supported design
system adapters (Hydro, shadcn, Material, Carbon, headless, vanilla). It is a
selection contract, not a bundled frontend runtime; customer auth, routing,
policy, and cache behavior remain explicit.

## Consequences

Frontend teams can bootstrap a consistent starter choice immediately while
keeping Compact deployments dependency-free. Browser implementation and visual
acceptance remain Client Edition work.

## Verification

Project generator tests and preset smoke verify both artifacts for every preset.
