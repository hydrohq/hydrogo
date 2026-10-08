# ADR-0358: Generated rich-text storage in Client Editions

- **Status:** Accepted
- **Date:** 2026-10-08
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0347](0347-versioned-public-generated-admin-runtime.md), [ADR-0354](0354-blueprint-derived-admin-activation.md), [ADR-0355](0355-aggregate-generated-resource-route-registration.md)

## Context

Generated Admin rich-text journeys had been verified in a fixture that wrote to
a synthetic PostgreSQL schema. The native Project Edition's generated resource
and migration path had separate CRUD coverage, but no rich-text resource. Its
standalone example server also omitted the embedded `/vendor` route used by the
shared Admin shell, so Quill rendered its frame without initializing its editor.
The production Core server already serves these embedded assets.

The Edition boundary must keep persistence and asset activation explicit. A
rich-text widget is presentation metadata; it must not create a new SQL type,
HTML rendering policy, migration executor, or authorization rule.

## Decision

- A Blueprint `text` field with `ui.widget: richtext` uses the existing
  generated `text` storage contract. Quill Delta JSON is submitted as a string
  and stored in the generated text column through the resource API/repository.
  Widget metadata does not grant HTML trust or bypass generated validation.
- Editions continue to activate the generated API/Admin and apply the reviewed
  generated migration explicitly. The generator never executes DDL or chooses
  an Edition policy.
- A standalone Edition that renders the shared Admin Studio must expose the
  embedded Core files under `/vendor/` from the public `hydrogo/static` asset
  package. The Core server already mounts this path; a separate Edition server
  mounts it once on its own `ServeMux`. Only files embedded by Core are served.
- Generated rich-text verification uses a native Edition resource and its
  generated SQLite/PostgreSQL migration, API, repository, and Admin routes.
  Synthetic fixture storage remains useful for isolated workflow composition
  tests and is not evidence of Edition migration behavior.

## Alternatives

- Keep rich-text verification only in an in-memory or synthetic-schema fixture:
  rejected because it does not exercise the generated Edition persistence
  path.
- Store Quill Delta in a new JSON-specific column type: rejected because the
  existing versioned generated `text` contract already handles the serialized
  document and both supported SQL dialects represent it consistently.
- Let the Blueprint register asset or content routes: rejected because route
  activation and static asset hosting belong to the Edition/Core composition
  root, not presentation metadata.
- Depend on a remote Quill CDN: rejected because it adds a runtime network
  dependency and conflicts with self-contained asset delivery.

## Consequences

### Benefits

- Rich-text storage is exercised through a real generated Edition table and
  API on SQLite and PostgreSQL.
- A standalone Edition's Quill control receives the same pinned, embedded
  vendor assets as the Core Admin.
- Widget metadata remains presentation-only; policy, API validation, and
  migration approval remain Edition-owned.

### Costs and risks

- A standalone Admin server must mount the embedded vendor asset route once.
- Quill Delta remains a serialized document string. Editions that publish it
  as HTML must add and test an explicit sanitization/rendering boundary.
- The Project Edition example remains a loopback-only demo with permissive
  policy; it is a composition fixture, not production authorization guidance.

## Scale impact

- **Compact:** one bounded static route and one generated resource table are
  added only to an Edition that explicitly includes this resource. No worker,
  cache, browser runtime, or additional database pool is created.
- **Standard:** the resource reuses the Edition's existing connection pool;
  the Quill file is served from the existing embedded asset bundle.
- **Distributed:** the route remains stateless; API identity, tenant scope,
  record authorization, and write conflicts continue to use Edition-owned
  request/service contracts.

## Database impact

- **SQLite:** the new example table contains a required `TEXT` body column. The
  generated migration metadata marks the operation additive/transactional.
- **PostgreSQL:** the same generated contract uses `TEXT` and is exercised by
  the real PostgreSQL Edition browser flow with a non-superuser role.
- **Future adapters:** a new adapter must preserve the generated text field
  round trip and pass the generated resource contract suite before activation.

The additive `generated_article` example migration has SQLite/PostgreSQL
checksums and compatibility metadata. It has no destructive down migration;
rollback retains the table or restores/forwards through an operator-reviewed
recovery. The test Edition database is disposable and is recreated for each
browser run.

## Cache impact

No cache keys, TTLs, invalidation, stampede behavior, or cache service changes.
The static asset route serves immutable embedded files; Admin HTML and records
remain private under the existing response policy.

## Deployment and compatibility

The example's migration command explicitly applies the new article table.
Older Edition binaries ignore the additive table and continue to serve the
existing resources; rolling back the example binary does not drop article
data. Production Editions must review their own migration ordering and backup
recovery before enabling an equivalent resource. Core deployments already
serve `/vendor/`; a standalone Edition must mount the route before exposing the
Admin shell. There is no asset download, external CDN, or worker lifecycle.

## Validation

- `hydro project check` validates the regenerated resource/helper output and
  generator lock.
- Project Edition Go tests and vet pass with the additional generated package.
- The real Chromium acceptance flow creates, formats, reads through the API,
  edits, reloads, and deletes an article through the generated Admin.
- The same test runs against the generated Edition migration on SQLite and
  PostgreSQL 18. It checks the embedded Quill asset response and the stored
  Quill Delta formatting. PostgreSQL uses a `NOSUPERUSER NOBYPASSRLS` role.
- Generator migration metadata, moduleguard, migrationguard, assetguard, and
  Edition importguards are required release checks.

## References

- [Admin Studio product audit and plan](../planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md)
- [Native generated Edition first CRUD validation](../validation/native-generated-edition-first-crud-2026-10-08.md)
- [Generated rich-text PostgreSQL fixture validation](../validation/generated-editorial-postgres-composition-2026-10-08.md)
- `examples/project-edition/article.blueprint.yaml`
- `extension/v1/adminstudio`
- `hydrogo/static`
