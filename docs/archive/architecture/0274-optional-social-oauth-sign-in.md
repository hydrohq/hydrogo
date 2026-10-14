# ADR-0274: Optional social OAuth sign-in

- **Status:** Accepted (browser flow); member/mobile provisioning remains a separate adapter gate
- **Date:** 2026-09-28
- **Related:** [ADR-0138: Provider-neutral OIDC boundary](0138-provider-neutral-oidc-boundary.md), [ADR-0139: Versioned mobile API authentication boundary](0139-mobile-api-auth-boundary.md), [ADR-0178: Tenant-scoped Admin identity and sessions](0178-tenant-scoped-admin-identity-and-sessions.md)

## Decision

- Google, GitHub, and Apple browser authorization-code flows are optional and
  configured only when credentials and callback URLs are provided. With no
  configured provider, no OAuth routes are registered.
- The Core client uses `net/http`, PKCE S256, a signed, short-lived HttpOnly
  state cookie, same-origin return paths, bounded responses, no redirects, and
  verified email claims. Apple ID tokens are signature-checked against its
  published JWKS and validated for issuer, audience, nonce, and time bounds.
- OAuth identities are linked to existing active administrator accounts by a
  provider-verified email or an existing provider subject. Core does not create
  administrator accounts or assign roles. Editions that need public member
  provisioning must inject a member-specific resolver and session adapter.
- Upstream access tokens are used only for profile lookup and are discarded.
  Hydro issues its own opaque session cookie after the identity resolver
  approves the account. Provider credentials and state-signing key are supplied
  through environment configuration and never written to the database.
- The SQLite/PostgreSQL mapping table is tenant-scoped. PostgreSQL enforces RLS;
  linked subject mappings are append-only through the service contract and
  removed when the local user is deleted.

## Consequences and release gates

Compact installs carry no provider dependency or route unless configured. The
browser adapter has automated token/state/PKCE and Apple signature tests. Before
claiming member mobile login, an Edition must implement member provisioning,
mobile token issuance/revocation, provider key rotation and real-provider
certification for each enabled provider. SDK clients do not receive provider
secrets.

The `HYDRO_OAUTH_<PROVIDER>_CLIENT_ID`, `CLIENT_SECRET`, and `REDIRECT_URL`
environment variables enable a provider. `HYDRO_OAUTH_STATE_KEY` must contain
at least 32 bytes. Apple's client secret is the provider-issued signed client
secret value.
