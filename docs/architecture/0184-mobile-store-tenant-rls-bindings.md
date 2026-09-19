# ADR-0184: Bind mobile device session stores to tenant transactions

- Status: Accepted (mobile device session stores bound; database pool audit completed)
- Date: 2026-09-20
- Related decisions: [ADR-0177](0177-tenant-scoped-member-mobile-sessions.md), [ADR-0180](0180-postgres-tenant-rls-enforcement.md), [ADR-0181](0181-auth-identity-rls-call-sites.md), [ADR-0182](0182-member-store-rls-call-sites.md), [ADR-0183](0183-forms-media-rls-call-sites.md)

## Decision

All operations across `internal/auth.MobileAuthService` and `modules/members.MobileAuthService` now execute within single-connection tenant-scoped transactions via `database.TenantTransactor` (`Within` and `WithinQuery`).
Direct pool queries and transactions that bypass tenant context or connection-level PostgreSQL `app.current_tenant_id` session variables are fully eliminated.
Token family revocation upon replay/reuse detection commits cleanly within the transaction boundary before returning security errors, preventing uncommitted rollback of revocation state.

## Consequences

- Mobile user and member device session issue, access validation, token rotation/refresh, session revocation, and device listing enforce transaction-bound PostgreSQL RLS policies.
- Fail-closed behavior is enforced when `RequireTenantScope` is enabled and no valid tenant context is present.
- Explicit tenant SQL predicates remain in place as dual-layer defense-in-depth.
- Completes Phase 1 audit and tenant transaction binding across all repository stores.
