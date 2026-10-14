# 0003: Social sign-in for both admin staff and members

- Status: Accepted
- Date: 2026-10-15

## Context

Admin staff could already sign in with Google, GitHub or Apple
(`internal/auth`'s `SocialOAuthHandler`), but that resolver only ever links a
verified provider identity to an existing, active admin or editor account —
it never creates one, which is correct for staff but wrong for public site
members, who are self-service. Members had no social sign-in at all.

## Decision

Split the provider-facing mechanics out of the admin handler into
`auth.OAuthFlow` (`internal/auth/oauth.go`): PKCE, the signed/time-boxed
state cookie, the authorization-code exchange, and Apple's `id_token`
verification (JWKS fetch, signature, issuer/audience/nonce/expiry). It knows
nothing about accounts or sessions. `SocialOAuthHandler` (admin) embeds it
and keeps its existing account-linking-only policy.

A new member-facing surface, `modules/members/transport.OAuthHandler`,
embeds the same `auth.OAuthFlow` with its own cookie name/path and redirect
targets (`/member/oauth/`, default return `/member/account`, cancel redirect
`/member/login?social=cancelled`), so the two flows' state cookies never
collide even though they run in the same process. It resolves identities
through `modules/members.OAuthResolver`, implemented by
`modules/members/sqlstore.OAuthStore` against a new
`member_oauth_identities` table (migration `202610150001`, mirroring
`auth_oauth_identities`'s shape but FK'd to `members`): find by identity,
else by email (linking it and marking the member's email verified), else
auto-provision a new active member (`password_hash` empty — there is no
password to set).

Routes: `GET /member/oauth/{provider}/start` and
`GET /member/oauth/{provider}/callback`, registered only when at least one
provider is configured.

Configuration reuses each provider's existing admin app credentials
(`HYDRO_OAUTH_<NAME>_CLIENT_ID`/`_CLIENT_SECRET`) and the shared
`HYDRO_OAUTH_STATE_KEY`, but requires its own
`HYDRO_MEMBER_OAUTH_<NAME>_REDIRECT_URL` per provider, since the provider
must route admin and member callbacks to different URLs. A provider with no
member redirect URL configured is simply not offered on the member surface.

The theme view model (`extension/v1.MemberAuthView`) gained
`OAuthProviders []MemberOAuthProviderView`; both themes (`corporate`,
`newsroom`) and the built-in fallback render a "sign in with ..." link per
configured provider and nothing when the list is empty, so sites without
social sign-in are visually unchanged.

## Consequences

- The token-exchange and Apple JWT verification code exists once, shared by
  both surfaces; a bug fix there (see the MySQL `email_verified_at` dialect
  fix applied to both `internal/auth/sqlstore` and
  `modules/members/sqlstore`) still has to be made in both resolvers, since
  each owns its own account-linking policy.
- Member email uniqueness is global (`members.email UNIQUE`), not
  per-tenant, which was already true before this change (`Service.Register`
  has the same constraint); multi-tenant deployments where two tenants'
  members can collide on email are a pre-existing limitation, not something
  this feature introduces or fixes.
- Apple requires `client_secret` to be a short-lived signed JWT that an
  operator must regenerate periodically; there is still no CLI helper for
  that (tracked as a gap, not blocking this change since Google/GitHub need
  no such rotation).
