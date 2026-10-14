# ADR-0361: Edition-provided generated Admin CRUD capabilities

- **Status:** Accepted
- **Date:** 2026-10-08
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0347](0347-versioned-public-generated-admin-runtime.md), [ADR-0353](0353-public-generated-admin-studio-shell.md), [ADR-0354](0354-blueprint-derived-admin-activation.md), [ADR-0355](0355-aggregate-generated-resource-route-registration.md)

## Context

The generated Admin handler filters collection navigation and routes through
Edition-owned `CanReadResource`, while create, update, and delete authorization
is enforced by each resource API. The shared UI did not receive the Edition's
basic CRUD capabilities, so it showed create, edit, and delete actions even
when the API would reject them. That made a permission denial discoverable only
after opening a form or submitting it.

Blueprint metadata cannot grant authorization. The Admin handler also cannot
infer a user's permissions from a presentation role or from generated field
metadata. The Edition already owns the policy used by the resource API and is
the only layer that can provide accurate per-request, per-record capabilities.

## Decision

- The v1 generated Admin options expose an optional Edition callback that
  returns `Create`, `Update`, and `Delete` capabilities for a collection and
  record. An empty record ID requests collection-level create access; a record
  ID requests record-level update/delete access.
- When supplied, the callback controls the matching basic CRUD affordances:
  create links/forms, record edit links/forms, and delete actions. Direct Admin
  create/update/delete routes enforce the same callback before forwarding a
  mutation to the resource API. Denied collections and records are not
  disclosed beyond the existing read policy; denied CRUD actions return 403.
- `CanReadResource` continues to decide whether the collection and its list
  are readable. Because the current record URL is an edit view, a record with
  read but no update capability remains visible in the list but its edit link
  and direct edit route are unavailable. A separate read-only document view
  requires a later decision and is not inferred here.
- Resource APIs remain the authorization authority. Editions must continue to
  enforce create/update/delete policy in the API for non-Admin callers and
  races. The Admin capability callback is not a replacement for API policy.
- Draft saves/transitions/discards, locale copy, and revision restore continue
  to use their existing dedicated callbacks. Localized content saves also
  require the base record's update capability.
- Omitting the optional callback preserves the v1 legacy presentation
  behavior. Existing Edition APIs still must enforce their own policies; new
  Edition compositions should provide the callback so the UI reflects those
  policies. An explicit zero-value capability set denies all basic CRUD.
- Blueprint manifests and generated database artifacts do not gain
  permissions. The callback is bound only by the active Edition at runtime.

## Alternatives

- Infer capabilities from Blueprint fields, role labels, or navigation
  visibility: rejected because schema and presentation data cannot grant
  authorization.
- Remove all actions from the shared Admin: rejected because it would disable
  CRUD for every Edition, including Editions with valid write policies.
- Treat hidden controls as the authorization boundary: rejected because direct
  HTTP requests and non-Admin API callers must still be denied by Edition
  policy.
- Make the callback mandatory immediately: rejected because it would break
  existing v1 Edition composition at registration time. The additive optional
  callback lets Editions adopt it without disabling existing API enforcement.

## Consequences

### Benefits

- Editors see only the basic CRUD actions their Edition policy grants.
- Direct Admin requests are checked consistently with the same capability
  source before the shared handler calls the API.
- Record-level update/delete visibility can differ within one collection.
- Existing Edition composition remains source-compatible; the API remains the
  final authorization boundary.

### Costs and risks

- Editions must map their policy model to a second, presentation-facing result
  so the UI can reflect API capabilities. Both must derive from the same
  request and permission rules.
- A stale or overly broad UI callback can display incorrect affordances, but
  cannot authorize an API mutation if the resource API remains correctly
  protected.
- The current generated document route combines read and edit presentation;
  a user without update access can see list fields but cannot open a read-only
  detail page.
- The optional callback preserves legacy behavior when omitted. Edition
  maintainers must adopt it to receive the improved permission-aware UI.

## Scale impact

- **Compact:** no service, route, asset, worker, or persistent resource is
  added. When configured, the callback runs once for a create affordance and
  at most once per displayed record for row actions; list size is bounded by
  the existing page limit.
- **Standard:** use the authenticated request's already-resolved policy state.
  Callback implementations should avoid per-row database queries; resource
  API policy remains unchanged.
- **Distributed:** callbacks remain request-scoped and stateless. Each Edition
  instance resolves the same principal/tenant policy as its resource API.

## Database impact

- **SQLite:** no schema, query, transaction, tenant, migration, or adapter
  behavior changes. Existing SQLite generated CRUD contracts remain valid.
- **PostgreSQL:** no schema, query, transaction, tenant, migration, or adapter
  behavior changes. Existing PostgreSQL generated CRUD contracts remain valid.
- **Future adapters:** no database behavior is introduced by this UI/runtime
  capability callback.

No migration metadata or rollback DDL is required because no persisted state
changes. Existing generated migrations retain their SQLite/PostgreSQL
compatibility and recovery contracts.

## Cache impact

No cache service, key, TTL, invalidation, stampede protection, or failure
fallback changes. The callback uses current request policy state and its result
is not cached across requests.

## Deployment and compatibility

The callback and result type are additive to the v1 options struct. Existing
Editions that omit the callback retain current UI behavior and existing API
authorization requirements. Editions that provide it must rebuild against the
updated v1 package; no route path, schema, asset, or persisted contract changes.
Blue/Green instances can run old and new binaries against the same database.
Rollback is a binary rollback; there is no migration or shared state to
recover. The sample Edition binds its permissive demo policy only in its
loopback-only fixture.

## Validation

- Handler tests cover all-denied capabilities, independent update-without-
  create/delete, hidden actions, direct-route 403 responses, and no API
  mutation after denial. Existing tests verify nil callback compatibility.
- The public facade test and standalone external Go Edition compile and use the
  new v1 result type.
- Chromium tests verify the read-only action state, valid-CSRF create/update/
  delete denials, and unchanged persisted data. The complete generated
  editorial browser suite passed 7/7 on both SQLite and PostgreSQL 18.
- SQLite and PostgreSQL generated Edition CRUD contracts remain unchanged; no
  database behavior is added by this decision.
- `go test ./...`, `go vet ./...`, moduleguard, migrationguard, and all Edition
  importguards remain required.

## References

- [Admin Studio product audit and plan](../planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md)
- [Admin permission matrix validation](../validation/admin-permission-matrix-2026-10-08.md)
- `extension/v1/adminstudio`
- `internal/adminstudio/generated_http.go`
