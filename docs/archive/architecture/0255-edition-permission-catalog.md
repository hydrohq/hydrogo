# ADR-0255: Edition-registered permission catalog

- **Status:** Accepted
- **Date:** 2026-09-23
- **Related decisions:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0198](0198-admin-user-account-and-permission-overrides.md), [ADR-0242](0242-resource-auth-composition-flag.md), [ADR-0243](0243-generated-auth-route-guard.md)

## Context

Generated resources declare names such as `article.read` and `article.write`,
but Admin Studio account overrides previously accepted and displayed only the
fixed Core permission list. This made generated policies impossible to
configure through the existing user permission surface and encouraged direct
role checks in Edition code.

## Decision

The Core permission set remains fixed. An Edition may compose a bounded
`auth.PermissionCatalog` from Core permissions plus the explicit sorted list
returned by generated `<Resource>Permissions()` functions. The generated
catalog contains names only: it does not select role defaults, grant access,
activate routes, or invent policy. New resource permissions are deny-by-default
for editor/viewer roles unless the Edition implements an explicit role policy
or an administrator assigns an individual override.

The catalog rejects invalid names and more than 512 unique entries; each
resource spec also has a 512-name bound. The permission-update form is capped
at 256 KiB, above the maximum URL-encoded form for 512 bounded ASCII names.
Only active catalog names may be assigned through the admin permission
endpoint. Session resolution filters stored overrides against the current
catalog. Custom policies use `PermissionCatalog.Allows` or `AuthService.Can`,
which deny names outside the active catalog even for an administrator. The generated auth
runtime wiring checklist requires this catalog to be installed before the
server begins serving requests.

The Admin Studio user editor and permission catalog view render the active
Edition list, escaped as text. Core's existing role defaults and static
permissions retain their behavior. No database schema change is required:
permission override rows are already tenant-scoped strings. Rows for a removed
permission remain stored but are ineffective while the name is absent from the
active catalog; reintroducing that same permission name requires a release
review because existing explicit overrides may become effective again.

This decision partially supersedes the static override-catalog provision in
ADR-0198. Account lifecycle, role management, last-admin protection, and
session-revocation rules remain unchanged.

## Consequences

- Generated resources can participate in the existing user-specific RBAC
  surface without introducing an Edition-owned auth implementation.
- The Core does not guess role grants or authorization semantics from CRUD
  permission names.
- Permission catalog composition is startup work only and introduces no SQL,
  cache, worker, or external-service overhead.
- The Edition remains responsible for registering every active resource's
  permission list and writing service-level policy tests.

## Verification

Tests cover bounded catalog construction, deny-by-default and unknown-name
behavior, rejection of unregistered writes, stale override filtering, generated
permission output, and escaped Admin Studio rendering. SQLite/PostgreSQL data
schema is unchanged; deployment permission behavior remains Edition-owned.
