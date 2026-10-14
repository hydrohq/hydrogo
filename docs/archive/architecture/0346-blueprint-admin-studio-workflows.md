# ADR-0346: Blueprint Admin Studio and editorial workflows

- **Status:** Accepted
- **Date:** 2026-10-07
- **Supersedes:** [ADR-0345](0345-generated-resource-admin-activation.md)
- **Superseded by:** None
- **Related decisions:** [ADR-0027](0027-provider-neutral-field-registry.md), [ADR-0209](0209-typed-server-ui-contract.md), [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0223](0223-generated-admin-ui-and-publishing-preview.md), [ADR-0229](0229-generated-list-transport-contract.md), [ADR-0231](0231-bounded-admin-saved-view-contract.md), [ADR-0234](0234-generated-admin-view-metadata.md), [ADR-0235](0235-generated-tenant-scoped-resource-contract.md), [ADR-0238](0238-global-singleton-database-invariant.md), [ADR-0239](0239-generated-global-admin-surface.md), [ADR-0240](0240-generated-route-composition.md), [ADR-0260](0260-generated-immutable-field-enforcement.md), [ADR-0261](0261-blueprint-create-default-presence.md), [ADR-0272](0272-admin-studio-business-ux.md)

## Context

Blueprints now produce a build-time resource registry and a shared Admin
handler. The handler is reachable by URL, but resources are not registered in
the Admin Studio navigation, and its basic list and form do not provide the
query, document, and editorial workflows expected from a CMS. The existing
generated Templ views are not the views used by that handler.

The target interaction reference is Payload CMS: grouped collection
navigation, searchable/filterable/sortable paginated lists, document editing,
draft and published states, revision history, and field-level localization.
HydroGo adopts those workflow ideas while retaining its Go server-rendered
Design System, module contracts, Client Edition ownership, and explicit
activation model. This decision does not add React or copy Payload's runtime.

Blueprint presentation metadata cannot grant access or define customer
business policy. Generated UI, APIs, workflow transitions, and migration
proposals must remain behind the Edition's composed authorization, tenant,
validation, audit, and data lifecycle boundaries.

## Decision

### Resource activation and navigation

- The generator emits deterministic, versioned resource metadata for collection
  labels, navigation group/icon/visibility, document title field, default list
  columns, and allowlisted searchable/filterable/sortable fields.
- The Edition composition root explicitly activates resources and binds their
  metadata to an already-composed resource API and Admin security boundary.
  Activation fails closed when a resource lacks its API, authorization, tenant
  resolver when required, CSRF boundary, or page renderer.
- Activated collections and globals appear in the existing Admin Studio
  navigation and use the existing shell, locale, and Design System. Hidden
  navigation metadata never grants route access.
- Collection list, create, edit, and global settings routes use a stable
  versioned route contract. The earlier `/admin/resources` surface remains a
  compatibility redirect during migration.

### List and document views

- Collection lists use the generated bounded query contract for search,
  filtering, sorting, and pagination. Field and operator allowlists, tenant
  scope, maximum query sizes, and resource policy are enforced by the resource
  API; the Admin handler does not query repositories directly.
- Default columns and a title field come from reviewed Blueprint metadata.
  Editors can follow a record into its document view, see validation errors at
  the field, and receive explicit save/conflict feedback.
- Forms use typed Templ Design System components. Relation, media, rich-text,
  array, and block controls are presentation adapters only; the application
  service revalidates all submitted values and ownership.
- The same document route supports generated globals as a single settings
  surface, without collection pagination or create/delete affordances.

### Drafts, revisions, and publication

- Revision history is opt-in per generated resource. When enabled, the
  generator proposes a version table and repository/service/API contracts;
  every persisted version records the document identity, monotonically
  increasing version, actor reference, timestamp, editorial state, and bounded
  schema-versioned snapshot.
- Drafts are opt-in and require revision history. Editing a published document
  as a draft must not change the published representation. Publish, unpublish,
  and restore are explicit service operations with expected-version checks,
  resource authorization, tenant scope, audit, and CSRF protection.
- Every draft save and workflow transition compares both the current draft
  version and the published resource version atomically. A stale published
  version blocks save or publish until the editor reloads; a new draft for an
  unpublished identity starts with published version zero.
- The Admin document view can browse revisions, compare bounded snapshots,
  and request a restore. Restore creates a new version; it never deletes or
  rewrites prior history. Autosave is separately opt-in, bounded, idempotent,
  and uses the existing allowlisted server-rendered enhancement contract.
- Editorial transitions remain resource/module-owned. The Admin Studio renders
  the registered workflow and does not infer permissions or transition rules
  from a status field or label.

### Localization

- Localization is opt-in per resource and field. The Edition owns the enabled
  locale set and locale access policy; the Admin document view selects an
  authorized content locale independently from the Admin interface language.
- Generated APIs and repositories preserve a stable document identity across
  locales and apply field validation and uniqueness rules to the correct
  locale scope. Copying content between locales is an explicit audited action.
- The migration proposal preserves existing rows as the Edition's configured
  default locale. It includes SQLite/PostgreSQL compatibility metadata,
  rollback/recovery notes, and blue/green compatibility requirements.

### Runtime and lifecycle

- Registry, routes, migrations, assets, and workflow work are activated only
  for resources and capabilities selected by the Edition. Unselected workflow
  options add no tables, routes, workers, or mandatory services.
- Blueprints only generate reviewable artifacts. They never execute DDL,
  register runtime routes, grant roles, pick tenant policy, or activate a
  background worker.
- Generated operations use the same application service and API contract as
  other callers. The Admin view never accesses SQL or cache drivers.

## Alternatives

- Copy Payload's React/Next.js admin runtime: rejected because it adds a
  frontend runtime and a second UI stack to HydroGo Core and Compact Editions.
- Keep the current direct-URL generic screen: rejected because it is not
  discoverable in Admin Studio and omits the requested editorial workflows.
- Put resource policy in Blueprint presentation metadata: rejected because
  schema visibility is not authorization and cannot define Edition policy.
- Store one unbounded JSON version blob or locale map in the main row: rejected
  because it weakens bounded history, query/index behavior, and portable
  migration review.

## Consequences

### Benefits

- A Blueprint resource has a discoverable and consistent Admin workflow after
  one explicit Edition activation point.
- Lists, forms, editorial state, and translations use shared contracts while
  policy and module-specific behavior remain owned by the Edition/resource.
- Optional version and locale storage does not burden resources that do not
  request those capabilities.

### Costs and risks

- Resource and generated API contracts require versioned compatibility rules.
- Version, draft, and localization migrations need certified SQLite and
  PostgreSQL behavior and Blue/Green review before an Edition enables them.
- Snapshot restore, concurrent editors, locale-specific validation, and
  relation/media controls increase the contract-test surface.

## Scale impact

- **Compact:** no browser framework or mandatory worker. History and locale
  storage exist only when selected. List, snapshot, and autosave payloads are
  bounded.
- **Standard:** use Edition database pool limits and bounded API pagination;
  no new cache is required.
- **Distributed:** handlers remain stateless. Authorization, tenant scope,
  workflow, and optimistic version checks are resolved through the Edition's
  shared service/API boundary.

## Database impact

- **SQLite:** generated dialect metadata and certified tests cover uniqueness,
  version ordering, draft isolation, restore, and locale constraints.
- **PostgreSQL:** the same logical contract is certified with PostgreSQL
  placeholders, transaction behavior, and Blue/Green migration compatibility.
- **Future adapters:** domain/service code remains dialect-neutral. New
  adapters must pass the same generated contract suite before activation.

## Cache impact

Admin responses remain private and non-cacheable. No new cache key, TTL,
invalidation, stampede behavior, or cache service is introduced.

## Deployment and compatibility

Migrations are generated proposals with dialect metadata and recovery notes;
they are never run by generation or startup. Expand/migrate/contract ordering
must allow old binaries to ignore new revision/locale tables while new binaries
continue to read existing default-locale documents. Rollback restores the
previous binary and follows the migration's documented forward-recovery path;
revision history is not silently discarded.

## Validation

- Generator tests verify deterministic navigation, query, workflow, locale,
  and migration artifacts, plus compatibility checks across resource sets.
- SQLite and PostgreSQL contract tests cover list queries, tenant isolation,
  draft/published separation, optimistic concurrency, revision diff/restore,
  locale isolation/copy, and rollback metadata.
- Admin handler and Edition composition tests cover navigation activation,
  policy denial, CSRF, field-level errors, empty/loading/failure states, and
  end-to-end create/edit/publish/restore/translate flows.
- Responsive desktop/mobile UI captures and accessibility checks verify the
  shared Admin shell, collection list, document editor, revision view, and
  locale selector.
- `moduleguard`, `migrationguard`, and `importguard` remain mandatory gates.

## References

- [Payload collection configuration](https://payloadcms.com/docs/configuration/collections)
- [Payload Edit View](https://payloadcms.com/docs/custom-components/edit-view)
- [Payload versions](https://payloadcms.com/docs/versions/overview)
- [Payload localization](https://payloadcms.com/docs/configuration/localization)
- `docs/architecture/ADR_TEMPLATE.md`
- `AGENTS.md`
