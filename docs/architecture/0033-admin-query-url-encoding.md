# ADR-0033: Stable admin query URL encoding

## Status

Accepted

## Decision

Admin links and forms use `EncodeListQuery` to serialize the shared list
contract. Empty values are omitted, filter keys are deterministic, and page
size remains bounded. This keeps browser adapters independent of repositories.

## Compatibility and capacity

The helper is additive and has no database or migration impact. Encoding is
bounded by caller input and performs no I/O, cache access, or background work.
