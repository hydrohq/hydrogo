# ADR-0363: Project Edition block content and live preview

- **Status:** Accepted
- **Date:** 2026-10-08
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0223](0223-generated-admin-ui-and-publishing-preview.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0358](0358-generated-richtext-edition-storage.md), [ADR-0360](0360-hash-authorized-block-editor-loader.md)

## Context

The generated block editor had been exercised with the official Editor.js
bundle in a synthetic browser fixture, but the Project Edition had no real
block resource, bundle activation, stored document contract, or Edition-owned
preview. The typed component's live preview also needed the block value and
the enclosing form's CSRF token while sending no unrelated form fields.

## Decision

- Project Edition adds a separate `content_block` Blueprint resource with a
  required title and a required `TEXT` field using the `block-editor` widget.
  Existing Article rich-text data and migration are unchanged.
- The reviewed generated resource creates the additive `content_blocks` table
  with SQLite/PostgreSQL compatibility metadata. The Edition migration command
  applies it explicitly; generation never applies DDL.
- The Edition owns and serves the pinned Editor.js 2.31.6 UMD asset at a
  same-origin path. Core does not gain a mandatory Editor.js asset or route.
- The sample Edition accepts only bounded Editor.js documents containing 1 to
  100 Paragraph blocks. It sanitizes each paragraph with the pinned
  `bluemonday` UGC policy before the resource API persists it. Unsupported
  block types, malformed JSON, and oversized documents fail closed.
- The Edition-owned preview route requires the demo CSRF token, accepts only
  one `body` value, validates and sanitizes the document, and returns fixed
  paragraph wrappers with `private, no-store` caching. The route remains part
  of the loopback-only example and is not production authentication guidance.
- When preview is configured, the typed `BlockEditor` emits fixed
  `hx-include="closest form"` and `hx-params="body,_csrf"` attributes. This
  allowlisted component contract sends only the block value and CSRF field;
  other forms and HTMX components gain no new behavior.
- The hash-authorized Admin shell loader fetches the same-origin HTMX bundle
  only when a generated block field has live preview configured, then processes
  the existing form. Editions without block preview do not load this runtime.
- The renderer returns HTML only after parsing the allowlisted block schema
  and sanitizing paragraph content. Caller-provided markup is never trusted.

## Alternatives

- Replace Article's existing Quill field with Editor.js: rejected because
  stored Delta documents would not be compatible with Editor.js documents.
- Keep the feature only in the synthetic browser fixture: rejected because it
  does not certify an Edition migration, route, asset, API, or preview path.
- Add Editor.js to Core: rejected because Editions that do not enable block
  content must not acquire the browser asset.
- Send the complete Admin form to preview: rejected because it exposes unrelated
  fields and needlessly enlarges the request.
- Render saved block text without sanitization: rejected because Editor.js
  paragraph data may contain untrusted inline HTML.

## Consequences

### Benefits

- A real generated Edition demonstrates automatic Admin registration,
  migration-backed persistence, editor asset loading, live preview, and CRUD.
- Existing Article data and Quill behavior remain compatible.
- CSRF and bounded block data travel through the typed preview component.

### Costs and risks

- The sample supports only Editor.js Paragraph blocks; other tools need an
  explicit schema, renderer, validation, and browser coverage before use.
- The Project Edition demo uses permissive policy and loopback-only binding;
  deployable Editions must replace its identity and tenant boundaries.
- `hx-include` and `hx-params` become part of the BlockEditor component
  contract and require component tests if changed.

## Scale impact

- **Compact:** only an Edition that includes the `content_block` resource
  receives the `TEXT` table and Editor.js asset. No Core worker, cache,
  background job, or new database pool is added. Editor.js and HTMX are fetched
  only on a form containing a block field with preview enabled.
- **Standard:** the resource reuses the Edition's existing DB pool; preview is
  bounded to 256 KiB of document data and runs synchronously in the request.
- **Distributed:** the preview route is stateless. A deployable Edition must
  resolve the same identity and tenant policy as its resource API.

## Database impact

- **SQLite:** generated migration creates `content_blocks` with `body TEXT NOT
  NULL`; the generated SQLite contract tests the same CRUD behavior.
- **PostgreSQL:** generated migration creates the same portable `TEXT` field;
  the generated PostgreSQL contract and browser flow certify persistence.
- **Future adapters:** adapters must preserve the serialized Editor.js text
  round trip and run the generated resource contract suite before activation.

The migration metadata declares additive/expand behavior and transactional
application for SQLite and PostgreSQL, with no destructive down migration.
Rollback retains the table and data; the previous binary can ignore the new
resource. Operators restore or apply a forward migration if a schema rollback
is required.

## Cache impact

Preview responses are private and non-cacheable. The fixed Edition asset path
uses a five-minute cache lifetime because it is not content-hashed. No
application cache or invalidation path is added.

## Deployment and compatibility

The new table and resource are additive. Old Project Edition binaries ignore
the table and continue to serve Article/Project/Milestone. Blue/Green binaries
may coexist because the existing resource contracts are unchanged and the new
resource is isolated. Deploy the Edition asset with the binary. Rollback does
not delete saved block data; restore or a reviewed forward migration handles
database recovery.

## Validation

- Generated resource contract tests verify the SQLite/PostgreSQL migration
  metadata and CRUD behavior.
- Edition tests verify the served official bundle, generated form wiring,
  CSRF-protected sanitized no-store preview, API create/read, rejection of
  unsupported block mutations, and unchanged state after rejection.
- Chromium uses the real Project Edition server and migration to create,
  preview, read, edit, and delete a Paragraph document.
- `go test ./...`, `go vet ./...`, moduleguard, migrationguard, Edition
  importguards, assetguard, and the PostgreSQL Chromium run are required.

## References

- [Admin Studio product audit and plan](../planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md)
- [Generated block editor validation](../validation/generated-block-editor-2026-10-08.md)
- `examples/project-edition/content-block.blueprint.yaml`
- `examples/project-edition/custom/composition.go`
- `internal/ui/components/block_editor.go`
