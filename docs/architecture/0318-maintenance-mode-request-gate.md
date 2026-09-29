# ADR-0318: Maintenance mode request gate

- **Status:** Accepted for the opt-in request-gate contract; scheduling and preview link provisions partially superseded by ADR-0321
- **Date:** 2026-09-29
- **Related decisions:** ADR-0013, ADR-0030, ADR-0286

## Decision

Maintenance mode is an optional server middleware contract. The Site module stores the enabled flag and IP/CIDR allowlist in a private, tenant-scoped setting exposed through an authorized Admin Studio form. The server gate renders a typed page and returns a no-store 503 with `Retry-After: 300`. Active admin and editor sessions, configured exact IP/CIDR entries derived only from `RemoteAddr`, and login, authentication, health, and static asset paths bypass the gate. Forwarded client address headers are not trusted. Admin pages show a persistent status banner while the setting is active.

The middleware is inactive unless a caller explicitly provides an enabled-state function. It adds no migration or worker requirement. The setting is queried per request so changes become effective immediately across application instances; this keeps behavior consistent without distributed cache invalidation. Scheduled windows and secret preview links are defined in ADR-0321. Other provisions in this ADR remain in force.

## Verification

Tests cover the 503 response, retry and cache headers, rendered body, static/login bypass, CIDR allowlisting, active staff role checks, private setting validation and tenant isolation, and the Admin Studio banner.
