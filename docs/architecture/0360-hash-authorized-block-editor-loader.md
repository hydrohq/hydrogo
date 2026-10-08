# ADR-0360: Hash-authorized generated block editor loader

- **Status:** Accepted
- **Date:** 2026-10-08
- **Supersedes:** [ADR-0359](0359-edition-owned-generated-block-editor.md)
- **Superseded by:** None
- **Related decisions:** [ADR-0223](0223-generated-admin-ui-and-publishing-preview.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0347](0347-versioned-public-generated-admin-runtime.md), [ADR-0350](0350-admin-studio-csp-style-and-script-hashes.md)

## Context

ADR-0359 bound generated block fields to an Edition-owned Editor.js asset, but
required Editions to provide a per-request CSP nonce for dynamic component
scripts. The shared `StudioPageRenderer` uses the Admin shell's static CSP hash
catalog and did not expose that nonce. As a result, the advertised shared shell
could not compose a block editor without another custom rendering boundary.

## Decision

- `FieldBlocks` and `widget: block-editor` render the typed `BlockEditor`
  component. The component emits only escaped markup and the configured
  same-origin script URL as data; it emits no inline script and needs no CSP
  nonce.
- The shared Admin shell owns a static, hash-authorized initializer. It lazily
  loads the Edition asset only when a block field is present, initializes
  Editor.js once, and keeps the hidden form value synchronized with bounded
  JSON returned by `editor.save()`.
- Edition composition must provide a same-origin Editor.js asset path. Runtime
  route registration rejects absent or unsafe paths. The Edition remains
  responsible for actually serving a compatible bundle and validating every
  submitted block value.
- Live preview is optional. If configured, its same-origin endpoint remains
  Edition-owned and must enforce authorization, tenant scope, CSRF, bounded
  request size, block validation, sanitized rendering, and private/no-store
  responses. Without it, the control sends no preview request.
- The versioned `extension/v1/adminstudio` contract exposes the Edition asset
  and optional preview path. Editions without block fields need no asset or
  route.

## Alternatives

- Require a nonce callback for every Edition: rejected because the public
  shared shell does not expose its hash-based CSP nonce and does not need one
  for same-origin external assets.
- Put Editor.js in Core: rejected because Editions without block fields must
  not acquire its asset.
- Assume a Core preview path: rejected because preview rendering policy and
  authorization belong to the Edition.

## Consequences

### Benefits

- The shared Admin shell and Edition-owned bundle compose without per-page
  inline scripts or nonce coordination.
- No Editor.js request occurs on pages that do not render block fields.
- Preview remains opt-in and cannot silently target a missing endpoint.

### Costs and risks

- The Edition must serve a compatible Editor.js asset at the configured path.
- The static shell initializer becomes part of the CSP hash catalog and must
  stay covered by shell and browser tests.

## Scale impact

- **Compact:** the initializer is static shell code; Editor.js is fetched only
  on an activated block form.
- **Standard:** the Edition serves a same-origin static asset; no worker, pool,
  or cache is added.
- **Distributed:** the asset may use the Edition's existing immutable asset
  layer; block preview remains stateless and tenant-scoped.

## Database impact

- **SQLite:** no schema or query change.
- **PostgreSQL:** no schema or query change.
- **Future adapters:** persistence remains behind the Edition API.

No migration is generated. Rollback restores the prior binary and asset; stored
block values are not transformed.

## Cache impact

Admin pages and preview responses remain private/no-store. The static bundle
uses the Edition's existing asset policy. No application cache is added.

## Deployment and compatibility

The Editor.js asset must be deployed before enabling the corresponding
resource. Old Editions without block fields need no asset or route. Blue/Green
uses the existing JSON value contract; no worker, schema migration, or data
recovery step is introduced.

## Validation

- Component tests verify escaped markup, configured same-origin script path,
  and no inline scripts or assumed preview route.
- Handler tests verify block fields reject missing and cross-origin asset paths.
- Admin shell tests verify the initializer is included in the generated CSP
  hash catalog and is called after full-page and SPA navigation.
- Browser acceptance tests must use an Edition-served compatible Editor.js
  bundle to verify create/edit JSON persistence and lazy asset loading.
- `go test ./extension/v1/adminstudio` and the standalone Edition module verify
  the public contract; moduleguard, migrationguard, and importguard remain
  mandatory.

## References

- [ADR-0359: Edition-owned generated Admin block editor](0359-edition-owned-generated-block-editor.md)
- [ADR-0350: Allow verified Admin Studio inline assets through CSP hashes](0350-admin-studio-csp-style-and-script-hashes.md)
- `internal/ui/admin_shell.go`
- `internal/ui/components/block_editor.go`
