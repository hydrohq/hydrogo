# ADR-0179: Tenant-scoped auth security tokens and MFA

- Status: Accepted (invite/recovery/MFA repository boundary; provisioning and PostgreSQL RLS open)
- Date: 2026-09-20
- Related decisions: [ADR-0178](0178-tenant-scoped-admin-identity-and-sessions.md), [ADR-0168](0168-host-based-tenant-resolution.md)

## Context

Invitation tokens, password-recovery tokens and MFA secrets/recovery codes are
high-value credentials. A token hash or user ID must not be usable across
tenants, even if the underlying user identifier is valid elsewhere.

## Decision

Migration `202609200002_auth_security_tenant_scope` adds tenant discriminators
and lookup indexes to invites, recovery tokens, MFA factors and recovery codes.
The auth SQL store uses the validated tenant context for create, consume, read,
password mutation and code-use operations when shared mode is enabled. Main
runtime wiring reuses the tenant-scoped admin `UserStore` for all these
boundaries.

## Consequences

- Cross-tenant invite/recovery/MFA token use resolves as invalid or unavailable;
  missing scope fails closed.
- Existing records remain in the empty namespace pending controlled backfill.
- PostgreSQL RLS, tenant provisioning and operational key rotation remain
  release gates outside this repository boundary.
