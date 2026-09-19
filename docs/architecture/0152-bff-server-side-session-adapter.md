# ADR-0152: BFF server-side session adapter

- **Status:** Accepted (bounded single-instance reference)
- **Date:** 2026-09-19
- **Related decisions:** [ADR-0021](0021-headless-browser-session-boundary.md), [ADR-0151](0151-bff-server-side-request-policy.md), [ADR-0142](0142-distributed-profile-certification-boundary.md)

## Decision

HydroGo provides `internal/security/bffsession.SessionStore` as the common
session boundary, with `Store` as the bounded single-instance reference
implementation and `internal/security/bffproxy.AuthHandler` as its
login/logout exchange. The store creates a random opaque browser
cookie, keeps the Core session cookie only on the server, stores only a hash
of the CSRF token, applies expiry and a maximum entry count, and supports
explicit revoke. Unsafe BFF methods use the same store as a constant-time
CSRF validator. The auth handler accepts a bounded form, calls only Core's
login/logout endpoints server-to-server, and never forwards Core `Set-Cookie`
to the browser.

The in-memory adapter is intentionally not a shared production session service.
A multi-instance BFF can select the shared `RedisStore` implementation behind
the same boundary, but still requires encrypted-at-rest/session-keyed storage,
eviction, failover, key rotation and privacy retention evidence. `Secure=true` and an explicit SameSite policy are required
for production browser cookies; local HTTP fixtures may opt into `Secure=false`.

## Consequences

- Browser cookies and bearer credentials cannot become Core credentials.
- Session memory and request abuse are bounded in the Compact/local profile.
- Distributed persistence, provider-specific login policy and staging browser
  E2E remain deployment responsibilities and release gates.
