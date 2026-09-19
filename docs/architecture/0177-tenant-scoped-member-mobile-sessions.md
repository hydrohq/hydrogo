# ADR-0177: Tenant-scoped member mobile sessions

- Status: Accepted (member mobile token boundary; admin auth identity remains open)
- Date: 2026-09-19
- Related decisions: [ADR-0173](0173-tenant-scoped-members-and-sessions.md), [ADR-0175](0175-tenant-scoped-commerce-checkout.md), [ADR-0168](0168-host-based-tenant-resolution.md)

## Context

The client application authenticates members with access and refresh tokens.
Token hashes and device families are bearer credentials, so a token lookup or
revocation operation must not cross the host-resolved tenant boundary.

## Decision

The existing `member_mobile_device_sessions.tenant_id` column from migration
`202609190009_members_tenant_scope` is now enforced by `MobileAuthService` in
shared mode. Issue, access validation, refresh rotation, family/device revoke
and device listing require a validated tenant scope and predicate both the
session and member rows. The single-tenant compatibility path remains
unchanged.

## Consequences

- A member token from tenant A is invalid in tenant B or without a scope.
- Refresh-token family revocation remains atomic and tenant-bound.
- Admin/browser `users`, `sessions`, invites, recovery and MFA identity stores
  still require a separate tenant contract before shared admin auth is ready.
