# ADR-0178: Tenant-scoped admin identity and sessions

- Status: Accepted (users and browser sessions boundary; invites/recovery/MFA open)
- Date: 2026-09-20
- Related decisions: [ADR-0177](0177-tenant-scoped-member-mobile-sessions.md), [ADR-0173](0173-tenant-scoped-members-and-sessions.md), [ADR-0168](0168-host-based-tenant-resolution.md)

## Context

Admin Studio authentication uses the `users` and `sessions` tables. In a
shared deployment, an email, user ID or session token must not resolve across
host-resolved tenants, even when the identifier is valid in another tenant.

## Decision

Migration `202609200001_auth_identity_tenant_scope` adds tenant discriminators
and lookup indexes to users and browser sessions. The auth SQL stores require a
validated tenant context in shared mode for create, lookup, count, activation,
session lifecycle and transaction-backed user creation. Main runtime wiring
enables the boundary when tenancy is enabled; legacy single-tenant callers
retain the unscoped compatibility path.

## Consequences

- Cross-tenant admin users and session tokens resolve as not found; missing
  scope fails closed.
- Existing records remain in the empty namespace until controlled backfill and
  tenant provisioning are delivered.
- Auth invites, recovery tokens, MFA factors and password mutations still need
  tenant predicates before shared Admin Studio authentication is complete.
