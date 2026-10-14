# ADR-0097: Generated SWR adapter

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Client Editions often use SWR instead of React Query. Rewriting the generated
fetcher/key wiring slows prototypes and creates inconsistent cache behavior.

## Decision

Each generated resource includes `swr.ts` with typed key/fetcher wiring and a
safe default that disables focus revalidation. The file imports no SWR package;
the Client Edition owns the dependency, authentication, authorization, and
mutation invalidation policy.

## Consequences

SWR projects can adopt generated resources immediately without adding runtime
dependencies to Core or Compact profiles. Browser integration and bundle-size
acceptance remain Client Edition tests.

## Verification

Generator tests verify the file, marker, compatibility artifact set, and the
generated fixture compilation.
