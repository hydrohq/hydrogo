# ADR-0104: Bounded provider-neutral Admin bulk contract

**Status:** Accepted  
**Date:** 2026-09-16

## Context

Generated frontend helpers exposed a bulk limit, but Admin Studio had no shared
request envelope for selection normalization. Each module could otherwise
parse duplicate, malformed, or unbounded IDs differently.

## Decision

`internal/adminstudio` provides `BulkRequest`, `ValidateBulkRequest`, and
`NormalizeBulkIDs`. Action names are bounded and safe, IDs are trimmed,
validated, deduplicated by rejection, and sorted deterministically, with a
default maximum of 500 records. The contract carries no authorization or
mutation behavior; modules retain policy, transaction, audit, and capacity
ownership and may select a lower limit.

## Consequences

Admin and API surfaces can share one predictable bulk envelope without loading
extra services or leaking storage details into Core. Actual bulk operations,
partial-failure semantics, and per-module permissions remain explicit follow-up
work in the owning modules.

## Verification

`go test ./internal/adminstudio -count=1` covers normalization, unsafe input,
duplicate IDs, and the hard upper bound.
