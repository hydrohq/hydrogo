# ADR-0365: Viewer role in the shared generated Admin shell

- **Status:** Accepted
- **Date:** 2026-10-09
- **Supersedes:** The Viewer/custom-renderer limitation in [ADR-0353](0353-public-generated-admin-studio-shell.md)
- **Superseded by:** None
- **Related decisions:** [ADR-0347](0347-versioned-public-generated-admin-runtime.md), [ADR-0353](0353-public-generated-admin-studio-shell.md), [ADR-0361](0361-generated-admin-resource-capabilities.md), [ADR-0362](0362-generated-admin-read-only-record-view.md)

## Context

The versioned shared Studio renderer accepted Admin and Editor identities but
rejected Viewer identities. Generated Admin already supports Edition-owned
read access, read-only lists and records, and separate CRUD capabilities. An
Edition could authorize a Viewer to read generated content but could not use
the shared shell to present that experience.

## Decision

- The v1 `adminstudio` contract exposes `PageRoleViewer` and the shared renderer
  accepts Admin, Editor, and Viewer presentation identities.
- A presentation role never grants access. The Edition's `Authorize`,
  `CanReadResource`, and `ResourceAccess` callbacks and generated API policy
  remain the authorization boundaries.
- An Edition may authorize Viewer access to selected generated resources and
  return zero CRUD capabilities. The generated handler then presents readable
  records without mutation controls and rejects direct mutation requests.
- Privileged built-in routes remain governed by their existing route
  authorization. Supporting a Viewer shell identity does not authorize those
  routes or register additional resources.

## Alternatives

- Require each Edition to provide a custom Viewer renderer: rejected because
  it duplicates the shared shell and makes read-only workflows harder to
  integrate.
- Infer read or write access from `PageRoleViewer`: rejected because identity
  presentation must not become authorization policy.
- Present Viewer as Editor to reuse the helper: rejected because it would
  misrepresent the authenticated identity and could expose Editor-only UI.

## Consequences

### Benefits

- Independent Editions can use the shared shell for read-only generated
  content without a custom renderer.
- The displayed identity remains accurate while Edition callbacks control
  collection and record access.
- Existing read-only detail behavior composes with the public shell.

### Costs and risks

- Edition authors must keep rendered CRUD capabilities consistent with API
  policy; API authorization remains mandatory for direct callers and races.
- The shell may render for a Viewer only when the Edition explicitly permits
  the generated Admin route.

## Scale impact

- **Compact:** no new route, service, worker, asset, browser runtime, or
  persistent resource is added.
- **Standard:** the renderer uses the existing authenticated request and
  embedded server-rendered shell.
- **Distributed:** the Edition resolves identity, resource access, and tenant
  scope per request; no shared mutable state is introduced.

## Database impact

- **SQLite:** no schema, query, migration, or adapter behavior changes.
- **PostgreSQL:** no schema, query, migration, or adapter behavior changes.
- **Future adapters:** no database behavior is introduced.

No migration or rollback metadata is needed because no persisted state changes.

## Cache impact

No cache keys, TTLs, invalidation, stampede protection, or fallback behavior
change. Authorization remains request-scoped and is not cached by the shell.

## Deployment and compatibility

The exported role constant and accepted renderer identity are additive to the
versioned v1 package. Existing Editions and binaries keep their current
authorization behavior. New binaries can render Viewer identities when an
Edition opts in. Blue/Green operation requires no data migration; rollback is
a binary rollback.

## Validation

- Public package tests render a Viewer identity through the shared shell and
  continue rejecting unknown roles.
- An independent Edition contract proves Viewer list/detail reads, hidden CRUD
  affordances, direct valid-CSRF mutation denial, and unchanged persisted data;
  Admin and Editor CRUD remain covered.
- Opt-in Chromium acceptance checks Viewer shell identity, read-only list and
  detail, hidden privileged navigation, direct mutation denial, unchanged data,
  axe, and mobile overflow.
  Actual browser execution remains an external listener-capable gate.
- SQLite contract tests and repository guards are required. PostgreSQL data
  behavior is unchanged, but generated Edition PostgreSQL acceptance remains a
  separate gate.

## References

- [Admin Studio product audit plan](../planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md)
- [Independent Edition Admin validation](../validation/external-edition-generated-admin-2026-10-09.md)
- `extension/v1/adminstudio`
