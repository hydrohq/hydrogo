# ADR-0287: Maintenance mode request gate

- **Status:** Accepted for the opt-in request-gate contract
- **Date:** 2026-09-29
- **Related decisions:** ADR-0013, ADR-0030, ADR-0286

## Decision

Maintenance mode is an optional server middleware contract. The owning module provides the per-request enabled state and a typed, tenant-aware HTML renderer; the server gate returns a no-store 503 with a bounded `Retry-After` value. Active admin and editor sessions, configured exact IP/CIDR entries derived only from `RemoteAddr`, and login, authentication, health, and static asset paths bypass the gate. Forwarded client address headers are not trusted.

The middleware is inactive unless a caller explicitly provides an enabled-state function. It adds no migration, worker, database, or cache requirement. The request gate does not itself define a scheduler, secret preview link, CLI shutdown command, or persistent admin toggle; those require a separate module contract and deployment behavior before they are represented as delivered.

## Verification

Tests cover the 503 response, retry and cache headers, rendered body, static/login bypass, CIDR allowlisting, and active staff role checks.
