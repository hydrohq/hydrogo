# ADR-0150: Member mobile authentication is isolated from staff mobile access

- **Status:** Accepted
- **Date:** 2026-09-19
- **Related:** [ADR-0025](0025-separate-frontend-members.md), [ADR-0139](0139-mobile-api-auth-boundary.md)

## Decision

The customer mobile application authenticates `members.Member` identities, not
Admin Studio `auth.User` identities. It uses a member-specific versioned API
under `/api/v1/mobile/member/*` and a separate device-session table whose
foreign key targets `members(id)`. Staff mobile credentials and member mobile
credentials are never interchangeable.

Member tokens are opaque, short-lived access tokens paired with single-use
refresh tokens and server-side revocation families. Raw tokens never enter
logs, audits, analytics, or cache keys. Browser member cookies are not accepted
as mobile credentials. Disabling a member revokes all of that member's mobile
device sessions.

## Release gates

Before public enablement: SQLite/PostgreSQL lifecycle and concurrent-refresh
contracts, member disable/revocation, rate-limit and audit tests, OpenAPI
baseline, client fixture, and real-environment threat/device-loss drills are
required. Email verification, reset, consent and deletion flows remain the
separate member-product gates from ADR-0025.
