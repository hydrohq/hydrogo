# ADR-0031: Provider-neutral editorial workflow

## Status

Accepted

## Decision

Admin surfaces share draft, scheduled, published, and retracted states with a
small transition matrix. Preview links are built only from same-origin paths;
modules own token signing, expiry, and authorization. Persistence, schedulers,
and cache invalidation remain module/application responsibilities.

## Compatibility and capacity

This is additive metadata logic with no migration and identical SQLite/
PostgreSQL behavior. URL construction is constant-time and creates no workers
or persistent state. Existing module-specific workflows can adopt it gradually.
