# ADR-0223: Generated Admin UI and Publishing Preview Boundary

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related decisions:** [ADR-0027](0027-provider-neutral-field-registry.md), [ADR-0209](0209-typed-server-ui-contract.md), [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0211](0211-declarative-module-resource-contract.md)

## Context

Blueprint-driven persistence and transport scaffolding still left repetitive
admin CRUD views to each Client Edition. Publishing Editions also need a
consistent block-editor and live-preview boundary without adding a mandatory
browser runtime to HydroCore or permitting untrusted HTML to cross the
server-rendered UI boundary.

## Decision

Blueprints may declare presentation-only metadata under `fields[].ui`:

- `list`, `searchable`, and `filterable` visibility hints;
- a bounded provider-neutral `widget` identifier.

`admin.icon` and `admin.group` are navigation metadata only. They do not grant
route access, create routes, or change migrations. The generator emits
`generated/<collection>/admin/list.templ` and `form.templ`, which call typed
Design System components and carry CSRF through the typed form boundary.

The Publishing Edition may opt into the typed `BlockEditor` component. Its
Editor.js bundle is an Edition-owned asset, not a Core dependency. Block data
is submitted as bounded JSON to a protected, CSRF-checked preview endpoint;
the endpoint may return HTML only from a renderer that validates the block
contract and sanitizes rich text. Preview responses are private and
non-cacheable.

## Consequences

- Standard CRUD screens become deterministic and reviewable without placing
  business policy in `generated/`.
- Unknown widgets fail closed at the registry boundary; new widgets require a
  contract and component before becoming shared vocabulary.
- Compact Editions do not acquire Editor.js assets unless a blueprint opts in.
- The generated templates contain no HTML comments and do not use arbitrary
  HTMX attributes; the shared component allowlist remains the enforcement
  point.
- The generated list/form files are scaffolds, not authorization policy or
  repository implementations. Edition composition must still bind tenant
  scope, permissions, validation and route middleware explicitly.

## Verification

`internal/generator` verifies deterministic list/form output, UI metadata
mapping, block-editor selection and absence of HTML comments. The typed UI
component tests verify escaping, CSRF rendering, bounded HTMX attributes and
Editor.js/live-preview markup. `internal/ui` verifies bounded preview payloads,
private no-store responses and redacted renderer errors.
