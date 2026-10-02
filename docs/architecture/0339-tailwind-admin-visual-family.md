# ADR-0339: "Aurora" admin visual family, authored with Tailwind

- **Status:** Accepted
- **Date:** 2026-10-02
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0041](0041-admin-visual-theme-modes.md), [ADR-0053](0053-semantic-ui-token-contract.md), [ADR-0335](0335-admin-nova-theme-mode.md), [ADR-0338](0338-standalone-tailwind-build-time-css.md)

## Context

ADR-0338 added a standalone, build-time-only Tailwind CLI, but kept it
strictly isolated to `custom/ui/tailwind/` - `internal/ui`, the four existing
admin visual families (`hydrogo`, `shadcn`, `workbench`, `nova`), and the
ADR-0053 semantic token contract were explicitly left untouched.

A follow-up request asked for a fifth, Core-shipped Studio visual family -
selectable from the same `#ui-family-select` control as the other four - that
is itself authored with real Tailwind utility classes, specifically so
AI-assisted design/authoring tools have an idiomatic Tailwind surface to work
in in Core Studio, not only in Edition-specific `custom/` pages. This is a
materially different request than ADR-0338's: it requires new markup in
`internal/ui`, not just a new isolated directory.

The existing four families are pure CSS re-skins of one shared, mode-agnostic
dashboard markup (`internal/ui/admin_workbench.go`'s `html/template`), scoped
via `html[data-ui-theme="..."]` selectors (ADR-0041, ADR-0335). That pattern
cannot produce a Tailwind-authored family, because the shared markup carries
semantic class names (`.wb-metric`, `.dashboard-stat`, ...), not Tailwind
utility classes. A Tailwind family needs its own markup. This was already
identified and accepted as the tradeoff before this ADR was written: the
alternative (retrofitting Tailwind into the shared markup) was rejected by
ADR-0338 itself as a repeat of the `internal/ui/admin_shell.go:2747-2756` bug
at much larger scale.

## Decision

1. A fifth visual family, **`aurora`**, is added to the Admin Studio family
   selector (`#ui-family-select` on `/admin/account`) and to every place the
   existing four families are enumerated: the pre-paint bootstrap script, the
   SPA re-apply logic, the visual-mode label map, and the family-select
   change handler in `internal/ui/admin_shell.go`. It is purely additive:
   none of the four existing options, their markup, or their CSS is modified.
2. Unlike the other four families, Aurora's dashboard body
   (`internal/ui/aurora/dashboard.tmpl.html`) is its own `html/template`
   source, authored directly with Tailwind utility classes (`tw:flex`,
   `tw:rounded-xl`, `tw:bg-slate-900`, ...), compiled by the ADR-0338
   standalone Tailwind CLI into `internal/ui/aurora/dist/aurora.css`
   (committed, embedded via `//go:embed`, exposed as `aurora.CSSText`). It
   does **not** use `ds.ButtonClass(...)`-style typed Design System helpers
   or the ADR-0053 semantic token names - that is the entire point of the
   family. This is a deliberate, scoped, versioned exception: this ADR *is*
   the governance for it. It licenses Tailwind utility markup only inside
   `internal/ui/aurora/`, not anywhere else in `internal/ui` or Core.
3. Following the exact display-toggle mechanism ADR-0335 introduced for Nova,
   **every** family's dashboard markup is always rendered into the page
   (`internal/ui/admin_dashboard.go`'s `RenderAdminDashboard` now executes
   `adminWorkbenchDashboardTemplate`, `adminNovaDashboardTemplate`, and
   `aurora.DashboardTemplate` unconditionally, in that order), and CSS alone
   decides which one is visible for the active `data-ui-theme` value
   (`internal/ui/admin_aurora.go`'s `AdminAuroraDashboardLayoutCSS`). No
   server-side mode detection, route, or request round-trip is introduced.
4. `aurora.CSSText` is concatenated into the same single inline
   `<style nonce="...">` block as `AdminCSS` and the other four families'
   CSS, in `internal/ui/admin_shell.go`. This directly engages the hard
   constraint ADR-0338 §4 raised ("Tailwind-authored pages must never load
   the compiled Tailwind stylesheet on the same page as AdminCSS... because
   Tailwind v4's common-named CSS custom properties would collide with the
   ADR-0053 semantic token names"). This ADR resolves that constraint for
   this one, controlled case rather than overriding it:
   - `internal/ui/aurora/input.css` builds from
     `tailwindcss/theme.css layer(theme) prefix(tw)` and
     `tailwindcss/utilities.css layer(utilities) prefix(tw)` only - **not**
     the full `tailwindcss` entrypoint. This excludes Tailwind's preflight
     (global reset) layer entirely, so nothing in the compiled output can
     reset or restyle an element outside `.aurora-dashboard`.
   - Tailwind v4's `prefix(tw)` renames every generated utility class to
     `tw:<utility>` (e.g. `.tw\:rounded-xl`) and every generated theme CSS
     custom property to `--tw-<name>` (e.g. `--tw-color-slate-900`,
     `--tw-radius-xl`). None of these names can collide with an unprefixed
     ADR-0053 token (`--radius-lg`, `--shadow-md`, `--bg`, ...).
   - `internal/ui/aurora/embed_test.go` certifies both of these properties
     (no preflight signature present, no unprefixed `--radius-`/`--shadow-`
     variable defined) as a standing regression guard, not just a one-time
     manual check.
5. The standalone Tailwind CLI remains build-time-only, per ADR-0338: the
   compiled `dist/aurora.css` is committed, regenerated by
   `scripts/build-aurora-tailwind.ps1` (mirrors `scripts/build-tailwind.ps1`),
   and `go build ./...` never requires the CLI to be installed.
6. The shell chrome (sidebar, topbar, shared cards/buttons/inputs) is
   re-skinned with Aurora's own teal/slate palette using the exact same
   mechanism `nova`/`shadcn`/`workbench` already use: plain CSS under
   `html[data-ui-theme="aurora"]` reusing the shared ADR-0053 semantic
   selectors and token names (`AdminAuroraCSS`, `AdminAuroraDensityCSS`,
   `AdminAuroraTypeCSS` in `internal/ui/admin_aurora.go`, modeled directly on
   `AdminNovaCSS`/`AdminNovaDensityCSS`/`AdminNovaTypeCSS`). This is
   deliberately **not** Tailwind: only the dashboard *body* is the documented
   Tailwind exception (Decision §2); the chrome stays on the semantic token
   contract so the whole shell is consistently re-skinned, the same way Nova
   re-skins its whole shell, without multiplying the Tailwind-authored
   surface beyond the one bounded exception this ADR grants.
7. This is framed as a generic, Core-shipped capability available to every
   Client Edition - like `nova` before it - not a behavior built for one
   customer, satisfying AGENTS.md's "never fork HydroCore for a specific
   customer need" rule. Any Edition using Admin Studio gets the option; none
   is required to use it.

## Alternatives

- **Re-skin the existing shared dashboard markup with Tailwind classes**
  (the ADR-0041/0335 pattern). Rejected: the shared markup's classes
  (`.wb-metric`, `.dashboard-stat`, ...) are not Tailwind utilities: there is
  nothing for a Tailwind build to compile against without first rewriting the
  shared markup itself, which would also restyle `hydrogo`/`shadcn`/
  `workbench`/`nova` and contradict ADR-0053.
- **Keep Aurora's compiled CSS out of the shared `<style>` block and load it
  only when `aurora` is the active family (a conditional `<link>` or a second
  request).** Rejected for this iteration: the family switch is a pure
  client-side, no-reload `data-ui-theme` attribute change (ADR-0041); loading
  a second stylesheet only on a client-side switch would need either an
  always-present but initially inert `<link>` (same net stylesheet-loading
  cost as what was chosen, with more moving parts) or a page reload on family
  change, which would regress the existing zero-reload switching behavior for
  every family, not just the new one.
- **Use the full `tailwindcss` import (with preflight) and scope it with a
  wrapper selector.** Rejected: Tailwind v4's preflight layer targets bare
  element selectors (`*`, `html`, `body`, `button`, `input`, ...) that cannot
  be scoped under a parent selector without hand-rewriting the layer, which
  defeats the point of using the standalone compiler.
- **Also author the shell chrome (sidebar/topbar) with Tailwind utility
  markup**, instead of the ADR-0053 semantic token contract. Rejected: the
  chrome markup is shared by all five families; forking it into Tailwind
  would mean either duplicating the whole shell layout a second time (same
  cost/risk this ADR already accepts once, for the dashboard body, but not
  worth paying twice) or retrofitting Tailwind classes onto the one shared
  chrome template, which would restyle `hydrogo`/`shadcn`/`workbench`/`nova`
  too and contradict ADR-0053. Plain CSS re-skinning of the shared chrome
  (same mechanism as the other four families) achieves full-shell parity with
  `nova` at much lower risk.

## Consequences

### Benefits

- Core Admin Studio gets a fifth, additive visual option with an idiomatic
  Tailwind authoring surface, without touching any existing family, route, or
  the ADR-0053 semantic token contract.
- The ADR-0338 "never load Tailwind CSS alongside AdminCSS" constraint now
  has a documented, tested, reusable resolution (prefix + preflight
  exclusion) for any future Core-shipped Tailwind surface that needs to
  coexist with the admin shell on the same page.

### Costs and risks

- A second, intentional styling vocabulary now exists inside `internal/ui`
  itself (not just under `custom/`): Tailwind utility markup in
  `internal/ui/aurora/`, semantic tokens everywhere else in `internal/ui`.
  This is bounded to that one directory and must stay that way; any PR that
  adds Tailwind utility classes to another `internal/ui` component outside
  `internal/ui/aurora/` is a new architectural decision requiring its own
  ADR, not an extension of this one.
- Aurora's dashboard is a second markup surface to keep in rough feature
  parity (not pixel parity - it is a distinct visual family by design) with
  the shared dashboard's data fields as `AdminDashboardData` evolves. Both
  templates are executed against the exact same data value in
  `RenderAdminDashboard`, so a compile-time `html/template` error will catch
  a renamed/removed field immediately; it will not catch an added field that
  Aurora simply doesn't surface yet.
- Aurora's shell chrome and its dashboard body are now styled by two
  different mechanisms (semantic tokens vs. Tailwind utilities). This is
  intentional and documented, but a future contributor restyling the chrome
  must remember to use `AdminAuroraCSS`'s token values, not Tailwind classes,
  to keep the one Tailwind exception bounded to the dashboard body.

## Scale impact

- Compact: no runtime impact - the asset is static text embedded in the
  binary and the dashboard markup is static Go template source; no new
  worker, route, or service.
- Standard: no change.
- Distributed: no change; the embedded asset and markup are part of the
  binary, no shared state or cache coordination.

## Database impact

- SQLite: N/A - no schema, query, or storage change.
- PostgreSQL: N/A
- Future adapters: N/A

## Cache impact

N/A - no server-side cache key, TTL, or invalidation is introduced. The
family preference lives only in `localStorage`, exactly like the existing
four families (ADR-0041, ADR-0335).

## Deployment and compatibility

Purely additive Go source change (new CSS/JS allowlist entries, a new
`internal/ui/aurora` package, one new `aurora.DashboardTemplate.Execute` call
in `RenderAdminDashboard`). No migration, no route change, no breaking change
to any existing family. Old and new binaries can coexist during a Blue/Green
rollout without any compatibility shim: a client without the `aurora` option
in its `<select>` markup simply never sets `data-ui-theme="aurora"`, and the
layout-toggle CSS's default state already hides `.aurora-dashboard` for every
other family.

## Validation

- `internal/ui/admin_areas_test.go`:
  - `TestAdminShellIncludesAuroraThemeMode` - mirrors
    `TestAdminShellIncludesNovaThemeMode`: family selectable, option present,
    bootstrap/label/SPA-router allowlists updated, rendered shell carries
    `data-ui-theme="aurora"`.
  - `TestAuroraChromeSharesSemanticMarkupContract` - mirrors
    `TestNovaModeSharesSemanticMarkupContract`: the shell chrome
    (sidebar/topbar/cards/buttons/inputs) reuses the shared ADR-0053 semantic
    selectors, giving Aurora full-shell parity with Nova.
  - `TestAuroraModeDoesNotShareSemanticMarkupContract` - the narrower
    exception: Aurora's dashboard-visibility *layout* CSS specifically does
    **not** restyle those shared selectors, since the dashboard body is the
    one deliberate Tailwind exception (Decision §2).
- `internal/ui/aurora/embed_test.go`:
  - `TestEmbeddedAuroraCSSIsPresent` - the compiled asset is non-empty.
  - `TestDashboardClassesAreCompiled` - a representative subset of the
    `tw:`-prefixed utility classes used by `dashboard.tmpl.html` has a
    matching compiled selector, guarding against the dead-class bug fixed at
    `internal/ui/admin_shell.go:2747-2756`.
  - `TestAuroraThemeVariablesAreNamespaced` /
    `TestAuroraCSSExcludesPreflight` - the two collision guards for loading
    this stylesheet alongside `AdminCSS` (see Decision §4).
- Existing theme-mode tests for `hydrogo`/`shadcn`/`workbench`/`nova`
  (`internal/ui/admin_areas_test.go`, `internal/ui/admin_dashboard_kits_test.go`)
  continue to pass unmodified, confirming no regression to the other four
  families.
- `go build ./...` and `go test ./internal/ui/... ./custom/...` pass.
- `scripts/theme-token-lint.ps1` only scans `themes/` and is unaffected
  (confirmed: same definition/reference counts as before this ADR).
- Regeneration command:
  `pwsh scripts/build-aurora-tailwind.ps1 -TailwindCli <path-to-standalone-binary>`
  (binary from https://github.com/tailwindlabs/tailwindcss/releases, asset
  `tailwindcss-<os>-<arch>[.exe]`).

## References

- ADR-0041, ADR-0053, ADR-0335, ADR-0338
- `docs/UI_DESIGN_SYSTEM.md`
- `internal/ui/admin_nova_dashboard.go` (display-toggle pattern this ADR
  reuses)
- `scripts/build-tailwind.ps1` (script style this ADR mirrors)
