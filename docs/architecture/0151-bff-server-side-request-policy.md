# ADR-0151: BFF server-side request policy

- **Status:** Accepted (scoped policy primitive)
- **Date:** 2026-09-19
- **Related decisions:** [ADR-0021](0021-headless-browser-session-boundary.md), [ADR-0147](0147-single-tenant-production-deployment-and-bff-model.md), [ADR-0148](0148-web-frontend-bff-and-cors-policy.md)

## Decision

HydroGo ships a small `internal/security/bffpolicy` primitive for a separate
web BFF. It accepts only versioned `/api/v1` paths, validates the configured
Core origin, rejects browser `Authorization`/proxy credentials, removes
browser origin/forwarding headers, and sets a Core cookie supplied by a
server-side session adapter. The browser's `Cookie` header is never reused as
the Core session.

The primitive does not implement a BFF session store, token exchange, CSRF
bootstrap, or deployment proxy. Those remain responsibilities of the frontend
deployment and require a staging E2E drill before ADR-0021 can move from
Proposed to Accepted.

## Consequences

- A BFF cannot accidentally turn a browser bearer token or cookie into a Core
  credential-forwarding channel.
- Internal/admin, metrics, health, webhook, and unversioned routes are outside
  the reference boundary.
- Deployments must provide a server-side session adapter and preserve Core's
  own mutation, CSRF, idempotency, audit, and payment policies.
