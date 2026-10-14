# ADR-0181: Bind auth identity lookups to tenant transactions

- Status: Accepted (auth identity call sites covered; other direct pool paths remain open)
- Date: 2026-09-20
- Related decisions: [ADR-0178](0178-tenant-scoped-admin-identity-and-sessions.md), [ADR-0180](0180-postgres-tenant-rls-enforcement.md)

## Decision

`UserStore` and `SessionStore` operations that previously used `*sql.DB`
directly now execute through `database.NewTenantTransactor`. This keeps the
PostgreSQL tenant GUC and the query on the same transaction/physical
connection, while preserving the existing SQLite repository contract. Existing
`*sql.Tx` methods remain available for larger auth workflows and inherit the
caller’s already-bound transaction.

## Consequences

- Shared-mode login, session validation, creation, revocation, expiry cleanup,
  and user activation are covered by the RLS binding boundary.
- Single-tenant mode explicitly disables the optional RLS predicate per
  transaction.
- This does not certify mobile/member stores or every handler that still owns
  a direct `*sql.DB`; those are tracked as the next call-site audit.
