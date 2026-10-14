# ADR-0292: Validated page block style presets

**Status:** Accepted  
**Date:** 2026-09-29  
**Related:** [ADR-0266](0266-design-system-blocks-and-interactions.md), [ADR-0290](0290-final-design-form-and-discovery-capabilities.md)

## Context

Editors need a small visual choice of background tone and vertical spacing for
existing page blocks. These are presentation hints shared by first-party
rendering and trusted edition themes; they do not change block meaning or
introduce arbitrary CSS.

## Decision

- Existing block JSON may carry optional `bg_color` (`default`, `muted`,
  `primary_tint`, `dark`) and `padding_y` (`normal`, `compact`, `spacious`).
- The page editor exposes typed select controls for these presets. Server-side
  block validation rejects every value outside the allowlists.
- First-party block output and trusted edition-theme output receive a typed
  Templ style frame when a non-default preset is selected. CSS changes only
  design tokens for the frame so nested components retain readable foreground
  colors in light and dark modes.
- Presets use existing JSON payload storage. No SQL migration, worker, cache,
  route, or additional asset is required. Older binaries and custom renderers
  can ignore these optional keys while preserving the block payload.

## Compatibility and deployment impact

SQLite, PostgreSQL, and MySQL storage contracts are unchanged because block
payloads are already bounded JSON. Existing records default to the current
design; rollback to an older binary ignores the new display-only fields and
continues serving content. Runtime cost is one bounded validation and, only
when selected, one small wrapper element per styled block. No capacity profile
or inactive module gains a persistent resource.

## Verification

Validation tests cover accepted and rejected values, editor tests cover select
rendering and form round trips, and renderer tests cover first-party and theme
fallback styling. Full verification runs `templ generate` and the complete Go
test suite.
