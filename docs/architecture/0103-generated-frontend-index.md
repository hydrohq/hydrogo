# ADR-0103: Generate a namespaced frontend entrypoint

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Resource generation emitted each TypeScript adapter separately, requiring a
developer to remember file paths and risking symbol collisions when several
adapters were imported together.

## Decision

Every generated resource includes `index.ts` with namespace exports for the
API, query, bulk, import/export, upload, React, SWR, HTMX, vanilla, and kit
surfaces. The entrypoint has no runtime dependency and preserves each adapter's
existing public contract.

## Consequences

New projects can import one stable resource entrypoint while choosing any
presentation adapter. Namespaces intentionally avoid collisions between helper
names; unused adapters are still opt-in at application bundling time.

## Verification

`go test ./internal/generator -count=1` checks that the entrypoint is emitted,
compatible output requires it, and all adapter namespaces are present.
