# ADR-0040: Generated typed capability metadata

## Status

Accepted

## Decision

Each generated resource includes `kits.ts`, a typed, deterministic list of
selected capability names. It is metadata only; implementations and runtime
activation remain Client Edition composition responsibilities.

## Compatibility and capacity

The artifact is additive and migration-free. Omitted kits generate an empty
tuple and no additional services, routes, workers, or assets.
