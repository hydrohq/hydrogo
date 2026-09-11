# ADR-0021: Headless browser session and cross-origin boundary

- **Status:** Proposed
- **Date:** 2026-09-10
- **Related decisions:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0015](0015-versioned-http-api.md), [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md)
- **Implementation status:** Credential-free published read CORS allowlist middleware and config validation are implemented; cross-origin browser mutation/BFF e2e acceptance gates are open.

## Context

A headless frontend can be a separate deployment. The current Core session uses a host-only, `HttpOnly`, `SameSite=Lax` cookie and a double-submit CSRF value. These values are suitable for a same-origin integrated UI; enabling credentialed CORS for a random third-party frontend origin expands the cookie, CSRF, and canonical host boundaries.

Just allowing a starter to use `fetch(..., { credentials: "include" })` is not a secure cross-origin session contract by itself. CORS allowlist, preflight, `Origin` validation, cookie domain, CSRF token bootstrap, preview cache bypass, and post-redirect checkout conversions must be designed together.

## Proposed decision

In the first headless delivery, two distinct access modes are explicitly separated:

1. **Public read:** Published `/api/v1` projections do not require credentials. CORS is off by default; if enabled, it works only for the configured origin allowlist with `GET`, `HEAD`, and explicit response headers. Credentials are not combined with `*`.
2. **Browser mutation:** Browser operations requiring a session, such as cart, checkout, account, and preview, go to Core via a same-site deployment or frontend BFF. A separate browser origin does not directly access the Core session cookie, and Core does not initially open credentialed CORS.

Admin Studio always remains on the Core origin. OIDC, public API keys, personal access tokens, or mobile tokens are outside the scope of this ADR; they require a separate authorization and data lifecycle decision.

## BFF contract

An independent storefront can use a BFF on its own domain instead of exposing the Core session cookie to the user's browser:

- The browser only sees the storefront/BFF session.
- The BFF uses an adapter specifically designed for tenant/deployment identity and user session in server-to-server calls to Core; it does not silently proxy the existing browser cookie.
- CSRF is enforced at the BFF's browser boundary; Core still enforces its mutation policy, idempotency, audit, and payment redirect validation in its own layer.
- The preview token is not used as a persistent cache key in the URL query; it is a short-lived, audience-bound, `Cache-Control: private, no-store` operation.

This document does not define the BFF token exchange implementation. The first experiment can only be proven with a same-site reverse proxy topology or a server-rendered BFF fixture.

## Security and operations impacts

- The `Origin` allowlist is explicit in the deploy config; wildcard, regex, or reflected origins are not accepted.
- CORS preflight responses carry an origin-aware `Vary: Origin, Access-Control-Request-Method, Access-Control-Request-Headers` in the cache.
- Public CORS headers are not applied to admin, auth, media-private, webhook, health/metrics, or mutation endpoints.
- The `compact` profile does not load new workers, caches, or browser credential services.
- Changing the allowlist in blue/green requires safe overlap between the old/new deployment and a rollback plan.

## Acceptance gates

This ADR is only accepted after proving the following:

1. An unknown origin cannot receive CORS headers on a public API response; an allowlisted origin receives the correct `Vary` header only on a published `GET/HEAD` call.
2. A cross-origin `POST` / credentialed request is rejected at both preflight and actual request.
3. A separate repository starter passes the login → cart → checkout redirect happy path e2e via same-site or BFF fixture.
4. A preview token, private media, or draft content does not leak into the cross-origin public cache.
5. SQLite/PostgreSQL and compact profile contracts pass without creating additional network or unbounded resource behavior.

## Alternatives

- **Credentialed CORS directly to Core:** reduces frontend code in the short term, but opens a new attack surface for cookie/CSRF and origin management.
- **Bearer token per client:** can simplify mobile/API usage; token issuance, scope, revocation, leak, and audit surface are a whole new product area.
- **Integrated frontend only:** preserves current security assumptions but fails to meet the independent frontend goal of ADR-0020.
