# ADR-0341: Admin family overhaul - remove workbench, isolate shadcn and hydrogo

- **Status:** Accepted
- **Date:** 2026-10-02
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0041](0041-admin-visual-theme-modes.md), [ADR-0053](0053-semantic-ui-token-contract.md), [ADR-0335](0335-admin-nova-theme-mode.md), [ADR-0338](0338-standalone-tailwind-build-time-css.md), [ADR-0339](0339-tailwind-admin-visual-family.md), [ADR-0340](0340-tailwind-styling-migration.md)

## Context

After ADR-0340 converted every admin Studio family's CSS to Tailwind
`@apply`-against-existing-selectors (a CSS-only migration, zero markup
change), a direct user request asked for three further, unrelated changes to
the admin visual-family system in one pass:

1. Remove the "workbench" family entirely. It turned out to carry no
   distinct visual identity of its own - see "Workbench's undocumented
   origin" below - and its selectable name was considered confusing next to
   "hydrogo".
2. Make "shadcn" actually look like shadcn/ui. Before this ADR, `shadcn`
   (like `nova` and the former `workbench`) was a pure CSS re-skin of one
   shared, mode-agnostic dashboard markup (`internal/ui/admin_workbench.go`'s
   `html/template`, ADR-0041) - it changed colors and borders but never used
   shadcn/ui's actual component conventions (the card recipe, a real data
   table, badges), because the shared markup's class names
   (`.wb-metric-strip`, `.dashboard-stat`, ...) have nothing to do with
   shadcn/ui's.
3. Redesign "hydrogo" as a genuine flagship theme, not a reskin. hydrogo had
   no scoped `html[data-ui-theme="hydrogo"]` CSS block at all before this
   ADR - it rendered through the shared base tokens the same way the old
   "workbench" family did. The two were, in effect, visually identical.

### Workbench's undocumented origin

Before writing the removal, this ADR's author re-read ADR-0041 in full to
avoid inventing a false "supersession" of a decision that never covered
workbench. The finding: **ADR-0041's actual decision text never mentions
workbench.** It describes exactly four modes - "HydroGo light, HydroGo dark,
shadcn light, and shadcn dark" - as the admin Studio's supported appearance
set. "workbench" first appears in [ADR-0335](0335-admin-nova-theme-mode.md)
(Nova's ADR) as already-existing, undocumented context ("The Studio admin
shell already supports three persisted visual families - `hydrogo`, `shadcn`,
`workbench`"). There is no ADR that introduces workbench as a decision in its
own right. This ADR therefore records workbench's removal directly, rather
than marking ADR-0041 "Superseded" for a decision it never actually made.

### Why isolate shadcn and hydrogo's dashboard body, not their full chrome

[ADR-0339](0339-tailwind-admin-visual-family.md) (Aurora) already established
a precedent for a family with its own Tailwind-authored markup: an isolated
Go package (`internal/ui/aurora`) with its own compiled stylesheet and its
own `dashboard.tmpl.html`, wired into `RenderAdminDashboard` alongside the
other families' dashboard templates. Critically, Aurora only replaced the
**dashboard body** - the shared sidebar/topbar chrome stayed on the one
shared markup, restyled via plain scoped CSS the same way every other family
already does.

Family selection in this admin shell is client-only: a pre-paint bootstrap
`<script>` reads `localStorage` and sets `data-ui-theme`/`data-theme` on
`<html>` before first paint; the server has no signal to branch on at
render time. Every family's dashboard-body markup is therefore rendered
into the page unconditionally, and a per-family CSS rule
(`html:not([data-ui-theme="X"]) .X-dashboard{display:none}`) hides the
inactive ones. Forking the shared sidebar/topbar chrome into per-family
markup would require applying the same unconditional-render-plus-CSS-toggle
treatment to the chrome on **every single admin page**, not just the
dashboard - a materially larger and riskier change than what was asked for
(shadcn and hydrogo's dashboard look and feel). This ADR deliberately scopes
"own Tailwind-authored markup for shadcn and hydrogo" to mean: their own
isolated dashboard-body package (exactly matching Aurora's precedent), plus
a properly shadcn/flagship-authored restyle of the *existing* shared chrome
markup via scoped CSS (the same mechanism every family's chrome restyle
already uses). This is the same kind of deliberate, documented exception
ADR-0340 already used for the public theme's five block-template exception.

Nova is explicitly left untouched by this ADR - it keeps its existing
from-scratch dashboard template (`internal/ui/admin_nova_dashboard.go`) and
shared-chrome-restyle pattern, since nothing about this request required
changing it.

## Decision

1. **Remove workbench entirely.** Deleted `internal/ui/admin_workbench.go`
   (its CSS consts and `adminWorkbenchDashboardTemplate`) and every
   reference to `"workbench"` across `internal/ui/admin_shell.go` (the
   pre-paint bootstrap script, `applyVisualMode`, `updateVisualModeLabel`,
   the family-select change handler, the theme-toggle `validModes` map, the
   CSS concatenation list) and `internal/ui/admin_areas.go` (the family
   picker's `<option value="workbench">`). `"hydrogo"` replaces
   `"workbench"` as the default/fallback family everywhere a fallback was
   needed - a deliberate choice, since hydrogo is now the flagship family,
   not an arbitrary substitution.

2. **shadcn gets its own dashboard-body package**
   (`internal/ui/shadcnui`), built the same way as Aurora: its own
   `input.css`/compiled `dist/shadcn.css` (standalone Tailwind CLI v4.3.3,
   `tw:`-prefixed, no preflight), its own `dashboard.tmpl.html`, its own
   `embed.go`, and contract tests mirroring Aurora's
   (`embed_test.go`: embedded CSS present, no preflight, no unprefixed
   `--radius-*`/`--shadow-*` theme-variable definition, no literal hex
   colors in markup, representative compiled selectors present). The markup
   uses real shadcn/ui component conventions: the card recipe
   (`rounded-xl border bg-card text-card-foreground shadow-sm`, expressed
   here as `tw:rounded-xl tw:border tw:border-[var(--line)]
   tw:bg-[var(--surface)] tw:shadow-sm`, reusing ADR-0053 tokens via
   `var()`), a real `<table>`/`<thead>` data table for the collections list
   with `rounded-full` count badges, and `gap-4`/`gap-6` spacing rhythm
   throughout. The shared chrome stays on `AdminShadcnCSS` (unchanged by
   this ADR), restyled via plain CSS exactly as before.

3. **hydrogo becomes the flagship family**, with its own distinct visual
   identity for the first time (`internal/ui/admin_hydrogo.go`'s
   `AdminHydroGoCSS`, scoped to `html[data-ui-theme="hydrogo"]`, plus its own
   dashboard-body package `internal/ui/hydrogoui`, built the same way as
   Aurora/shadcn). Design executed:
   - **One confident brand hue**, a cobalt blue (`#2954ff` light-mode primary,
     `#6b89ff` for dark-mode contrast) - deliberately not a purple-blue
     gradient, not teal, not monochrome.
   - **Soft real elevation**: `--shadow-sm`/`--shadow-md`/`--shadow-lg`
     overridden with a brand-tinted ambient shadow (e.g.
     `0 10px 28px -8px #2954ff1f,0 3px 8px #1014250d`) instead of flat
     (`shadow:none`, as shadcn uses) or hard-offset neobrutalist shadows.
   - **Confident shape**: slightly more generous corner radii than the
     shared default (`--radius-lg:18px`, `--radius-xl:28px` vs. the base
     `16px`/`24px`).
   - **Light-first with a proper dark companion**: both palettes defined
     together, not a mechanical invert.
   - **A genuinely restructured dashboard body**, not a reskinned KPI row:
     a welcome header card, icon-badge stat cards (each with a colored icon
     chip, not just a bare number), a collections grid, and a two-column
     system-status + quick-actions panel.

4. **Dashboard-body visibility toggle.** `RenderAdminDashboard` now executes
   all four dashboard templates unconditionally (`hydrogoui`, `shadcnui`,
   `nova`, `aurora`), same pattern as before. hydrogo deliberately has **no
   self-hiding CSS rule** - like the "workbench" family it replaces, it is
   the default/fallback and stays visible unless another active family
   explicitly hides it. `AdminShadcnDashboardLayoutCSS` (new) gives shadcn
   its own self-hide-unless-active rule plus an explicit
   `.hydrogo-dashboard{display:none}` when shadcn is active (mirroring
   Aurora/Nova's existing pattern). `AdminNovaDashboardLayoutCSS` and
   `AdminAuroraDashboardLayoutCSS` were updated to hide `.hydrogo-dashboard`
   instead of the now-deleted `.wb-dashboard`.

## Alternatives

- **Keep workbench as an alias of hydrogo instead of deleting it.** Rejected:
  the user explicitly asked for removal, and keeping two selectable names
  for one visual identity is the exact confusion being fixed.
- **Fork shadcn/hydrogo's chrome markup, not just the dashboard body.**
  Rejected for now - see "Why isolate shadcn and hydrogo's dashboard body,
  not their full chrome" above. Nothing about the user's request required
  diverging the sidebar/topbar structurally, and doing so would require
  extending the client-only unconditional-render-plus-CSS-toggle pattern to
  every admin page, not just the dashboard.
- **Give hydrogo a non-blue signature hue (amber, emerald, deep purple).**
  Considered and explicitly decided against via user confirmation in favor
  of a cobalt blue, to avoid both the "purple-blue gradient" AI-slop pattern
  and a collision with Nova's existing green "Aktif" status badges.

## Consequences

### Benefits

- One less confusing, undifferentiated family name in the picker.
- shadcn and hydrogo now have genuine, distinguishable visual identities
  instead of three families (`hydrogo`/`shadcn`/`workbench`, pre-ADR) sharing
  one undifferentiated base look.
- The Aurora isolated-package pattern is now validated twice more
  (shadcn, hydrogo), confirming it generalizes beyond a single "fifth,
  additive" family.

### Costs and risks

- This is the second deliberate exception (after ADR-0339's Aurora) to
  ADR-0041's "all families render through byte-identical shared markup"
  invariant. Three of five families (`shadcn`, `hydrogo`, `aurora`) now
  carry their own dashboard-body markup; only `nova` and the shared chrome
  itself remain on the fully shared-markup model.
- Three isolated Tailwind packages (`aurora`, `shadcnui`, `hydrogoui`) now
  need their own `scripts/build-*-tailwind.ps1` re-run whenever their
  `dashboard.tmpl.html` classes change; forgetting this produces dead
  unmatched classes, caught by each package's
  `TestDashboardClassesAreCompiled` contract test, not by the compiler.
- hydrogo's lack of a self-hiding CSS rule is an implicit invariant (it must
  always be the one family without a `display:none`-unless-active rule, or
  no-JS sessions will see a blank dashboard). This is the same implicit
  invariant workbench carried before; it is now documented here instead of
  being undocumented a second time.

## Scale impact

- Compact: no change - three more static files embedded via `//go:embed`
  into the single binary; no new services, workers, or routes.
- Standard: no change.
- Distributed: no change.

## Database impact

- SQLite: none.
- PostgreSQL: none.
- Future adapters: none - this is a UI-only change.

## Cache impact

None. `AdminHydroGoCSS`, `shadcnui.CSSText`, and `hydrogoui.CSSText` are
Go string constants/embedded files concatenated into the existing
`<style>` tag at render time, under the same no-cache-key render path the
rest of `internal/ui`'s CSS consts already use.

## Deployment and compatibility

Standard blue/green rollout. No stored data, session, or API contract
changes. A client with the old build's JS running against a new-build page
(or vice versa) still works: `localStorage`'s `hydro-ui-theme-v2` value for
a session that previously had `"workbench-light"`/`"workbench-dark"` stored
falls through the family allowlist check (`['hydrogo','shadcn','nova',
'aurora'].includes(parts[0])`) to the `"hydrogo"` default on next load - no
migration script needed beyond the existing fallback logic.

## Validation

- `internal/ui`: `TestWorkbenchFamilyIsFullyRemoved`,
  `TestAdminShellIncludesHydroGoThemeMode`,
  `TestHydroGoDashboardRendersFlagshipWidgets`,
  `TestShadcnDashboardRendersDataTableWidgets`,
  `TestAdminShellHidesInactiveDashboardVariantByCSS` (updated).
- `internal/ui/shadcnui`, `internal/ui/hydrogoui`: each package's
  `embed_test.go` - embedded CSS present and non-empty, dead-class guard,
  theme-variable namespacing guard, no-preflight guard, no-literal-hex-color
  guard, plus a design-intent guard
  (`TestShadcnDashboardUsesCardRecipe`,
  `TestHydroGoDashboardIsRestructuredNotAReskin`).
- `go build ./...` and `go test ./...` both green.
- `go run ./tools/migrationguard`, `go run ./tools/assetguard` both green.
  `go run ./tools/moduleguard` reports pre-existing, unrelated findings
  (`modules/members`, `modules/news`, `modules/pages`, `modules/site` ->
  `modules/designsystem`/`seo`/`site`), confirmed present before this
  change and untouched by it.

## References

- [ADR-0041](0041-admin-visual-theme-modes.md)
- [ADR-0335](0335-admin-nova-theme-mode.md)
- [ADR-0338](0338-standalone-tailwind-build-time-css.md)
- [ADR-0339](0339-tailwind-admin-visual-family.md)
- [ADR-0340](0340-tailwind-styling-migration.md)
