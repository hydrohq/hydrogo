# ADR-0335: Admin "Nova" theme mode and family selector

- **Status:** Accepted
- **Date:** 2026-09-30
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0041](0041-admin-visual-theme-modes.md), [ADR-0053](0053-semantic-ui-token-contract.md)

## Context

The Studio admin shell already supports three persisted visual families —
`hydrogo`, `shadcn`, `workbench` — scoped via `html[data-ui-theme="..."]`
(ADR-0041) and built on a shared semantic token contract (ADR-0053). A
customer wanted a fourth, shadcn/Vercel-dashboard-inspired look, added
strictly as an additional option: none of the existing families may be
removed or modified.

While wiring this in, we found that the shipped Studio UI has no working
control that actually selects a family: the only functioning toggle button
flips the `light`/`dark` variant. A family is only ever set by a one-time
legacy migration to `workbench`, or by editing `localStorage` by hand. Adding
a fourth family the same way would leave it practically unreachable, so this
decision also introduces the first real family-selection control.

## Decision

1. A fourth visual family, `nova`, is added following the exact ADR-0041
   pattern: token blocks under `html[data-ui-theme="nova"]` and
   `html[data-ui-theme="nova"][data-theme="dark"]`, reusing the same semantic
   token names defined by the base `AdminCSS` tokens (`--bg`, `--surface`,
   `--ink`, `--brand`, `--radius-*`, `--shadow-*`, etc. — ADR-0053), with its
   own indigo/slate palette. Structural component rules (sidebar, topbar,
   cards, buttons, inputs, dashboard grid) are scoped the same way the
   `workbench` family already scopes them, so the existing, unmodified
   dashboard markup (`internal/ui/admin_workbench.go`'s `html/template`,
   already mode-agnostic) renders correctly without any Go template changes.
2. A topbar `<select id="ui-family-select">` is added, listing all four
   families. Selecting one sets `data-ui-theme` on `<html>`, keeps the
   current light/dark variant, and persists the choice the same way the
   existing light/dark toggle does (`hydro-ui-theme-v2` / `hydro-ui-theme` in
   `localStorage`). The existing light/dark toggle button is unchanged.
3. The pre-paint bootstrap script, the SPA-navigation re-apply logic, and the
   visual-mode label updater are all extended to recognize `nova` alongside
   the existing three families; none of their behavior for `hydrogo`,
   `shadcn`, or `workbench` changes.
4. The admin login page is intentionally left out of scope: it already does
   not participate in the family system (it has its own independent token
   set), so Nova does not need to either.

## Alternatives

- **Restyle the dashboard markup itself** instead of adding a CSS-only mode.
  Rejected: the existing markup is already mode-agnostic and reusable; a
  markup fork would duplicate the dashboard template per mode, which ADR-0041
  and the "never fork for a specific need" contribution rule both argue
  against.
- **Leave family selection as localStorage-only** (no picker). Rejected: a
  mode nobody can reach through the running UI is not a shippable feature.

## Consequences

### Benefits

- A fourth, additive visual option is available without touching any
  existing mode's CSS, tokens, or behavior.
- Closes a pre-existing usability gap: all four families (including the
  previously unreachable `shadcn`) are now selectable from the running UI.

### Costs and risks

- One more CSS token/component surface to keep in sync if the shared
  semantic selectors (`.studio-card`, `.button`, `.studio-search`, etc.)
  change shape in the future; covered by the contract test described below.

## Scale impact

- Compact: N/A — pure client-side CSS/JS, no new service, route, or worker.
- Standard: N/A
- Distributed: N/A

## Database impact

- SQLite: N/A — no schema, query, or storage change.
- PostgreSQL: N/A
- Future adapters: N/A

## Cache impact

N/A — no server-side cache key, TTL, or invalidation is introduced; the
preference lives only in the browser's `localStorage`, exactly like the
existing three families.

## Deployment and compatibility

Pure additive Go source change (new CSS/JS string constants, new markup, new
allowlist entries). No migration, no route change, no breaking change to any
existing mode. Old and new binaries can coexist during a Blue/Green rollout
without any compatibility shim, since clients without the new `<select>`
markup simply keep using the existing light/dark toggle.

## Validation

- `internal/ui/admin_areas_test.go`: `TestAdminShellIncludesNovaThemeMode` and
  `TestNovaModeSharesSemanticMarkupContract`, mirroring the existing
  `TestAdminShellIncludesShadcnThemeModes` / `TestAdminShellIncludesWorkbenchThemeMode`
  / `TestHydroAndShadcnModesShareSemanticMarkupContract` coverage.
- Existing theme-mode tests for `hydrogo`/`shadcn`/`workbench` continue to
  pass unmodified, confirming no regression to the other three families.
- `go build ./...` and `go test ./internal/ui/...`.

## References

- `docs/ADMIN_UI_THEMES.md`
- ADR-0041, ADR-0053
