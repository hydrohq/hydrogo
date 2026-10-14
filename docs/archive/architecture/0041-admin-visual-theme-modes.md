# ADR-0041: Admin visual theme modes

- **Status:** Superseded by [ADR-0338](0338-standalone-tailwind-build-time-css.md) for the "no Tailwind" consequence only; superseded by [ADR-0341](0341-admin-family-overhaul.md) for the "every family renders through byte-identical shared markup" invariant, specifically for the `shadcn` and `hydrogo` families (their dashboard body is now its own isolated Tailwind-authored package; `nova` remains on the shared-markup model this ADR describes, unchanged); the two HydroGo/shadcn appearance modes below remain Accepted and unchanged
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md), [ADR-0037](0037-strict-modular-monolith-boundaries.md), [ADR-0341](0341-admin-family-overhaul.md)

## Context

Studio users have different visual preferences and accessibility needs. The
existing HydroGo light/dark appearance must remain available, while teams that
already use the shadcn/ui visual language need a familiar alternative without
adding a runtime frontend dependency or replacing the server-rendered panel.

## Decision

The Admin Studio supports four persisted appearance modes: HydroGo light,
HydroGo dark, shadcn light, and shadcn dark. The existing HydroGo rules and
tokens remain the default and are not removed. The shadcn mode is a local CSS
token/component layer selected with `data-ui-theme="shadcn"`; it uses the same
semantic markup, CSP nonce flow, keyboard behavior, and responsive layout.
The preference is stored only as an admin UI preference (`hydro-ui-theme`),
with the legacy `hydro-theme` key retained for variant compatibility.

## Consequences

- No React, CDN, or third-party runtime is introduced. (Tailwind: see
  [ADR-0338](0338-standalone-tailwind-build-time-css.md) — a standalone,
  build-time-only Tailwind CLI is now permitted, strictly isolated to
  `custom/ui/tailwind/`; it is not introduced here, in `internal/ui`, or in
  any of the four admin visual families below.)
- Existing routes and custom HydroGo styling remain intact and can be restored
  by cycling the appearance control.
- Visual parity is covered by token/markup contract tests; browser viewport
  certification remains part of the open E2E gate.
