# ADR-0030: Client address trust boundary

- **Status:** Accepted
- **Date:** 2026-09-12
- **Related decisions:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0024](0024-optional-redis-rate-limiting-and-queue-boundary.md)

## Decision

Rate limits and session audit metadata use a single request-metadata helper.
It removes the ephemeral port from the directly connected peer address. It does
not trust `Forwarded`, `X-Forwarded-For`, or similar headers until a configured
and tested trusted-proxy allowlist is introduced.

## Consequences

A caller cannot bypass a per-IP limiter by opening a new source port, nor can
it select a limiter key with a forged forwarding header. This changes no stored
schema and has no cache, worker, route, or inactive-module cost. Deployments
behind a reverse proxy are intentionally keyed by that proxy until a future
trusted-proxy configuration and its deployment contract are implemented.
