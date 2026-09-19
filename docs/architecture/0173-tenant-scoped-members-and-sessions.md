# ADR-0173: Tenant-scoped members and sessions

- Status: Accepted (member repository boundary; device-session and identity provisioning open)
- Date: 2026-09-19
- Related decisions: [ADR-0150](0150-member-mobile-auth-boundary.md), [ADR-0168](0168-host-based-tenant-resolution.md), [ADR-0172](0172-tenant-scoped-site-data.md)

## Context

The mobile client authenticates members and persists sessions. A host-resolved
tenant context must therefore reach member lookup, administrative search,
status changes and session lifecycle operations; otherwise a valid member ID or
token can be replayed against another tenant.

## Decision

Migration `202609190009_members_tenant_scope` adds tenant discriminators and
lookup indexes to members, member sessions and mobile device sessions. The
member and session SQL stores require a validated context scope in shared mode
for creation, lookup, search, stats and lifecycle mutations. Existing rows are
kept under the empty namespace until a controlled backfill is performed.

## Consequences

- Browser/mobile member identity and session operations fail closed without
  tenant context.
- Existing global email/token uniqueness is retained as a conservative
  compatibility constraint until tenant-aware identity provisioning is shipped.
- Mobile device-session repository methods, auth user/session stores and
  tenant-aware onboarding still require a follow-up contract.
