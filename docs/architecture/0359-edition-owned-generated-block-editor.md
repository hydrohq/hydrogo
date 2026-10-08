# ADR-0359: Edition-owned generated Admin block editor

- **Status:** Superseded
- **Date:** 2026-10-08
- **Supersedes:** None
- **Superseded by:** [ADR-0360](0360-hash-authorized-block-editor-loader.md)
- **Related decisions:** [ADR-0223](0223-generated-admin-ui-and-publishing-preview.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0347](0347-versioned-public-generated-admin-runtime.md), [ADR-0350](0350-admin-studio-csp-style-and-script-hashes.md)

## Context

Blueprint metadata already accepts `blocks` fields and the `block-editor`
widget, but the shared generated Admin form rendered them as plain text. The
typed `BlockEditor` component assumes an Edition-owned Editor.js asset and CSP
nonce, while its old default preview URL pointed to a route that may not exist.
This allowed an Edition to activate a field that looked supported but could
not work safely.

## Decision

- Generated Admin recognizes `FieldBlocks` and `widget: block-editor` as the
  typed `BlockEditor` presentation adapter.
- An Edition that activates either field must supply a same-origin Editor.js
  asset path and a per-request CSP nonce callback. The handler rejects
  registration when either is missing and fails closed if a request has no
  nonce. The Editor.js bundle remains Edition-owned and is loaded only by a
  resource that opts into the control.
- Live preview is optional. If configured, the same-origin preview endpoint is
  Edition-owned and must enforce authorization, tenant scope, CSRF, bounded
  request size, block validation, sanitized rendering, and private/no-store
  responses. Without a configured URL, the component emits no preview request
  or assumed route.
- Block JSON remains untrusted input and must be validated by the Edition's
  application service on every mutation. The Admin component never writes to a
  repository or decides which block types are allowed.
- The versioned `extension/v1/adminstudio` contract exposes this configuration;
  existing Editions without block fields need no changes.

## Alternatives

- Render block fields as a text input: rejected because it neither edits nor
  communicates the structured block contract.
- Add Editor.js to HydroGo Core: rejected because Compact and Editions without
  publishing blocks must not acquire the asset.
- Assume `/api/preview` exists: rejected because preview routes belong to the
  Edition and must have explicit security and rendering policy.

## Consequences

### Benefits

- Blueprint block fields render through the existing typed component.
- Missing Edition assets and CSP configuration fail during composition rather
  than producing a broken editor at runtime.
- Editions can omit preview without a phantom network request.

### Costs and risks

- Editions with block fields must serve a compatible Editor.js bundle and
  provide their CSP nonce.
- Preview behavior remains Edition-specific and needs its own endpoint tests.

## Scale impact

- **Compact:** no bundle or route is added unless an activated resource opts in.
- **Standard:** the Edition serves one static asset; the handler adds no worker,
  database connection, or cache.
- **Distributed:** the static bundle may be served by the Edition's normal
  asset layer; preview handlers remain stateless and tenant-scoped.

## Database impact

- **SQLite:** no schema or query change.
- **PostgreSQL:** no schema or query change.
- **Future adapters:** block persistence remains behind the Edition API.

No migration is generated. Rollback is a normal binary/asset rollback because
the contract only binds existing field values and does not transform storage.

## Cache impact

Admin pages and preview responses remain private/no-store. The static asset may
use the Edition's existing immutable asset policy. No application cache is
added.

## Deployment and compatibility

An Edition must deploy its Editor.js asset before enabling the corresponding
resource. Older Editions without block metadata remain compatible. Blue/Green
rollout retains the same JSON value contract; rollback restores the prior
binary and asset. No migration or background worker is needed.

## Validation

- Component tests prove escaped markup and absence of a default preview route.
- Handler registration tests prove block fields fail closed without asset and
  nonce configuration.
- Generated Admin browser tests cover create/edit persistence with an
  Edition-served compatible editor bundle when one is configured.
- `go test ./extension/v1/adminstudio` and the standalone external Edition
  module verify the versioned public contract.
- `moduleguard`, `migrationguard`, and `importguard` remain mandatory.

## References

- [ADR-0223: Generated Admin UI and Publishing Preview Boundary](0223-generated-admin-ui-and-publishing-preview.md)
- [ADR-0346: Blueprint Admin Studio and editorial workflows](0346-blueprint-admin-studio-workflows.md)
- [ADR-0347: Versioned public generated Admin runtime contract](0347-versioned-public-generated-admin-runtime.md)
- `internal/ui/components/block_editor.go`
