# ADR-0198: Admin user accounts and permission overrides

- **Status:** Partially superseded by [ADR-0255](0255-edition-permission-catalog.md) for the permission-catalog/override scope; account and role decisions remain accepted.
- **Date:** 2026-09-20
- **Related decisions:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0025](0025-separate-frontend-members.md), [ADR-0178](0178-tenant-scoped-admin-identity-and-sessions.md), [ADR-0179](0179-tenant-scoped-auth-security-tokens.md), [ADR-0255](0255-edition-permission-catalog.md)

## Decision

Admin Studio accounts are managed under `/admin/users`; frontend customer/member
accounts remain under `/admin/members` and use the separate member identity
boundary. Account owners may change their own email or password only after
verifying the current password. The operation revokes all browser sessions.

Administrators may invite, disable, and change the role of other Admin Studio
accounts. Role changes and explicit permission changes revoke the target's
sessions. The last active administrator and the current administrator cannot be
demoted through this surface.

Roles provide bounded defaults (`admin`, `editor`, `viewer`). A user-specific
permission override is an explicit `allow` or `deny` stored in the
tenant-scoped `admin_user_permissions` table; no row means inherit the role
default. The effective decision is evaluated by Core authorization code, not by
navigation visibility. Core permissions are fixed; generated resource
permissions are added only by the active Edition's explicit permission catalog.
Unknown or inactive-catalog permission names are rejected. No role grant is
inferred for a generated permission.

## Consequences and release gates

- SQLite and PostgreSQL receive the same additive migration and tenant
  discriminator. No external queue, translation service, or unbounded cache is
  introduced.
- Passwords are hashed with the existing Argon2id boundary and never logged.
- The initial UI covers account, role, active state, and override editing. Each
  module must migrate direct role checks to the permission contract before its
  permission is advertised as customer-configurable.
- Migration checksum, SQLite/PostgreSQL store tests, authorization tests, and
  a blue/green compatibility check are required before production promotion.
