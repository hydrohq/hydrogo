# ADR-0209: Typed server-rendered UI contract

- **Status:** Accepted
- **Date:** 2026-09-21
- **Scope:** Workspace-generated Client Editions

## Context

AI-generated server-rendered views tend to fall back to untyped HTML and
client-side assumptions. That creates inconsistent accessibility, accidental
HTMX behavior, and duplicated escaping rules. HydroGo must generate views
without adding a browser runtime to Core or the Compact profile.

## Decision

HydroGo provides a small typed Templ-compatible Design System contract under
`internal/ui/components`. Generated views use typed `Button`, `Input`, `Card`,
and `Form` properties rather than assembling arbitrary markup. HTMX attributes
are represented by an explicit allowlist (`hx-get`, `hx-post`, `hx-target`,
`hx-swap`, `hx-trigger`, and `hx-confirm`). Values are escaped at the component
boundary and forms can render the CSRF hidden field centrally.

The component package has no JavaScript or CSS runtime dependency. Client
Editions may add visual tokens and compiled assets, but Core does not embed a
frontend framework as a result of using this contract.

## Consequences

- Generated UI has a stable, testable security and accessibility boundary.
- Adding an HTMX attribute requires an explicit contract change and test.
- Existing legacy `html/template` handlers are not silently rewritten; migration
  is incremental and belongs to affected Editions/modules.
- The contract does not replace authorization, tenant scope, CSRF validation,
  or idempotency checks in handlers.

