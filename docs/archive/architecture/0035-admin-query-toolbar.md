# ADR-0035: Shared admin query toolbar

## Status

Accepted

## Decision

Admin list pages may render a shared GET toolbar for search, named filters,
sort, and bounded page size. Action paths are same-origin only; modules supply
allowed options and still apply repository allowlists.

## Compatibility and capacity

The renderer is additive, dialect-neutral, and requires no migration. It
performs bounded template work only and introduces no workers, cache keys, or
provider dependencies.
