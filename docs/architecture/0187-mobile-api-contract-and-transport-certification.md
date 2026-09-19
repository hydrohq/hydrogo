# ADR-0187: Mobile API contract and transport certification

- Status: Accepted
- Date: 2026-09-20
- Related decisions: [ADR-0139](0139-mobile-api-auth-boundary.md), [ADR-0150](0150-member-mobile-auth-boundary.md), [ADR-0177](0177-tenant-scoped-member-mobile-sessions.md), [ADR-0184](0184-mobile-store-tenant-rls-bindings.md)

## Context

Mobile client applications for HydroGo require secure, header-driven Bearer authentication across two distinct identity boundaries:
1. Admin & editorial user sessions (`/api/auth/mobile/*`)
2. Customer & member identity sessions (`/api/members/mobile/*`)

Both surfaces require certified token pair issuance, cryptographic family rotation, immediate token family revocation upon replay detection, remote device session revocation, device listing, and multi-tenant isolation.

## Decision

We formally certify the Mobile API contract and transport suite:
1. Header-only Bearer authentication is enforced; browser cookie mutation is rejected for mobile endpoints.
2. Single-use refresh token rotation: each refresh issues a new access/refresh pair and invalidates the previous refresh token.
3. Family replay detection: presenting a revoked refresh token immediately invalidates all active sessions in the token family across the database, persisting the revocation atomically before returning security errors.
4. Active user/member enforcement: deactivated or disabled users/members are prevented from refreshing or validating mobile access tokens.
5. Multi-tenant isolation: tokens and device sessions are strictly bound to their parent tenant scope; cross-tenant validation or refresh attempts are rejected fail-closed.

## Consequences

- Certified mobile client API contracts across both SQLite and PostgreSQL.
- Multi-tenant shared SaaS security guarantees are fully verified for mobile clients.
