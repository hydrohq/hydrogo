# ADR-0138: Provider-neutral OIDC boundary

- **Status:** Accepted (contract); provider integration remains an explicit release gate
- **Supersedes:** None

## Context

OIDC/SAML changes identity trust, token lifecycle, account linking, and tenant authorization. A provider-specific SDK in Core would increase coupling and make Compact deployments carry unnecessary dependencies.

## Decision

- Core exposes an issuer-neutral identity result: issuer, subject, email-verification state, groups/roles, and authentication time.
- Discovery, JWKS caching, nonce/state validation, redirect allowlists, clock skew, logout/revocation, and account-link policy belong to an adapter.
- Browser sessions remain opaque Hydro sessions; an upstream ID token is never used as a session cookie or logged.
- OIDC is opt-in and unavailable routes/dependencies are absent when disabled.

## Release gate

Before claiming OIDC support, certify at least one provider, key rotation, nonce replay rejection, issuer/audience validation, clock skew, logout/revocation, and SQLite/PostgreSQL auth transaction behavior.
