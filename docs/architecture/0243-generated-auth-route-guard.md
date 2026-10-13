# ADR-0243: Auth-enabled generated routes require an Edition checker

- **Status:** Accepted
- **Date:** 2026-09-22
- **Scope:** Generated `runtime.go` route composition for `auth: true`
- **Related:** ADR-0013, ADR-0240, ADR-0242

## Decision

When a resource declares `auth: true`, its generated `Register<Name>Route`
helper requires an Edition-supplied `func(*http.Request) bool` authentication
checker. Registration fails closed when the checker is nil, and the registered
handler returns `401` before the generated policy/transport for unauthenticated
requests. The checker is an adapter to the existing Identity/Session/RBAC
stack; it is not a credential store or a new auth implementation.

Resources without `auth: true` retain the existing registration signature and
behavior. Authorization, tenant resolution, CSRF, session refresh, audit, and
rate limiting remain owned by the Edition/module composition and outer server
middleware.

## Consequences

- Auth intent now has a concrete fail-closed runtime effect rather than being
  documentation-only metadata.
- Existing unauthenticated public resources remain source-compatible.
- The Core does not guess how an Edition authenticates users.

## Verification

Generator tests assert the auth-enabled signature, nil guard, and `401` guard;
non-auth generated examples continue to compile and pass their CRUD tests.
