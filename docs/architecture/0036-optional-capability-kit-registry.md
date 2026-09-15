# ADR-0036: Optional capability kit registry

## Status

Accepted

## Decision

HydroGo exposes a provider-neutral `KitRegistry` for localization, media, SEO,
forms, redirects, search, and commerce capabilities. Editions opt in to kits;
the registry contains metadata only and never imports or starts a kit's
implementation.

## Compatibility and capacity

The registry is additive and has no migration impact on SQLite/PostgreSQL. An
inactive kit creates no routes, workers, assets, or mandatory resource
overhead, preserving the Compact profile invariant.
