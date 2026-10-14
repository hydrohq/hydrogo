# ADR-0362: Generated Admin read-only record view

- **Status:** Accepted
- **Date:** 2026-10-08
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0361](0361-generated-admin-resource-capabilities.md)

## Context

ADR-0361 lets an Edition expose a readable collection while denying update
capability for a particular record. The shared Admin kept that record visible
in the list but returned 403 when the user followed its title. That prevented
read-only editors from inspecting content they are allowed to read.

## Decision

- When a non-draft collection record is readable but the Edition's
  `ResourceAccess` callback denies `Update`, its existing record URL renders a
  read-only detail view.
- The view includes only fields marked visible in the collection list, with
  escaped display values,
  uses the typed Design System card and link, and returns to the collection
  list. It emits no edit form, delete form, workflow or revision mutation
  controls.
- Existing POST routes continue to check `ResourceAccess` and deny mutations
  before calling the resource API. The API remains the authorization boundary.
- Draft collections retain their existing workflow-specific route and policy
  behavior. Global settings continue to require `Update` because their record
  URL represents the settings editor and they have no collection list view.
- `CanReadResource` remains the gate for collection and record visibility.
  No fields or values are added to the API contract; detail data is fetched
  using the same API read already used by the edit view.

## Alternatives

- Return 403 for read-only records: rejected because it hides readable content
  from read-only editors and makes title links confusing.
- Add a separate route and URL: rejected because the existing record URL can
  safely select its presentation using the same Edition capability callback.
- Render disabled form controls: rejected because the edit form implies a
  mutation path and complicates accessible read-only presentation.

## Consequences

### Benefits

- Read-only editors can inspect allowed records without receiving edit access.
- Existing links keep working while direct mutation requests remain denied.
- All rendered values are escaped and use the established server-rendered UI.

### Costs and risks

- The callback must agree with API read and update policy. Correct API-side
  read authorization remains mandatory.
- Field presentation uses the registered display values; specialized media or
  relation previews are not introduced by this view.

## Scale impact

- **Compact:** no services, routes, assets, workers, or persistent resources
  are added. The detail request performs one existing record API read.
- **Standard:** the policy callback is request-scoped and should use already
  resolved policy state.
- **Distributed:** no shared mutable state or cross-instance dependency is
  introduced.

## Database impact

- **SQLite:** no schema, query, transaction, migration, or adapter behavior
  changes.
- **PostgreSQL:** no schema, query, transaction, migration, or adapter
  behavior changes.
- **Future adapters:** no database behavior is introduced.

No migration or rollback metadata is required; persisted state and resource API
contracts are unchanged.

## Cache impact

No cache behavior changes. Record data and capability results are fetched or
resolved per request and are not cached by the generated Admin.

## Deployment and compatibility

The route and resource contracts are unchanged. New binaries render the detail
view for read-only users; existing binaries retain their prior 403 behavior.
Blue/Green operation is safe because no stored state or migration is added.
Rollback is a binary rollback.

## Validation

- Handler tests prove readable detail rendering, exclusion of non-list fields,
  escaping, absence of forms, and continued POST denials before API mutation.
- Chromium acceptance proves the read-only record opens with no forms and
  create/update/delete POST requests remain denied with valid CSRF.
- Existing SQLite and PostgreSQL contracts remain unchanged; focused and full
  Admin test suites and required repository guards must pass.

## References

- [Admin Studio product audit and plan](../planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md)
- [Admin permission matrix validation](../validation/admin-permission-matrix-2026-10-08.md)
- `internal/adminstudio/generated_http.go`
