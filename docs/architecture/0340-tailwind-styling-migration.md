# ADR-0340: Tailwind utilities as the admin Studio styling language

- **Status:** Accepted (partially delivered - see Validation and
  "Remaining work")
- **Date:** 2026-10-02
- **Supersedes:** [ADR-0339](0339-tailwind-admin-visual-family.md)'s "isolated
  exception, not a precedent for Core components generally" framing only -
  Aurora itself, its markup, and its Tailwind toolchain are unaffected and
  remain Accepted. ADR-0338's standalone-CLI decision is unaffected and not
  superseded.
- **Related decisions:** [ADR-0041](0041-admin-visual-theme-modes.md),
  [ADR-0053](0053-semantic-ui-token-contract.md),
  [ADR-0338](0338-standalone-tailwind-build-time-css.md),
  [ADR-0339](0339-tailwind-admin-visual-family.md)

## Context

ADR-0338 introduced the standalone Tailwind CLI strictly for new, isolated
UI work (`custom/ui/tailwind/`). ADR-0339 extended it once more, strictly to
Aurora's own dashboard body, explicitly framed as a one-off, documented
exception that "doesn't licence ad-hoc Tailwind elsewhere."

The user has since asked, after being warned of the architectural risk and
explicitly accepting it, to make Tailwind the styling language for **all**
of admin Studio: the shared shell markup and all four token-driven visual
families (hydrogo, shadcn, workbench, nova), not just Aurora. This is a
materially different decision from ADR-0338/0339 and needs its own record,
because it changes Tailwind from "isolated exception" to "the normal way
Studio's component CSS is authored" - while keeping every other guarantee
those two ADRs established (build-time only, no runtime dependency, no
markup fork, ADR-0053 token contract unchanged).

## Decision

Tailwind utilities, compiled by the same standalone CLI (ADR-0338), become
the authoring language for admin Studio's **component CSS** - the hand-written
rules that give meaning to the ADR-0053 semantic tokens (`.studio-card`,
`.nav-item`, `.button`, `.studio-sidebar`, etc.). This explicitly does **not**
change the shared-markup architecture ADR-0041 established:

1. **The shared HTML stays exactly as it is.** hydrogo/shadcn/workbench/nova
   continue to render through the same markup in `internal/ui/admin_shell.go`
   (plus `admin_workbench.go`'s and `admin_nova_dashboard.go`'s dashboard
   templates). No family gets its own forked copy of the shell. This is the
   one hard constraint that makes "convert everything to Tailwind" compatible
   with "never fork for a specific need" (AGENTS.md) and with the existing
   `TestHydroAndShadcnModesShareSemanticMarkupContract` /
   `TestNovaModeSharesSemanticMarkupContract` tests, which assert the four
   families share the same semantic selectors.

2. **Each family's token-definition blocks are untouched.** The
   `html[data-ui-theme="X"]{--bg:...; --surface:...; ...}` and
   `html[data-ui-theme="X"][data-theme="dark"]{...}` custom-property blocks
   (ADR-0053's token contract) remain hand-written, literal CSS - they are
   *values*, not styling logic, and Tailwind has no utility for "define a
   custom property." This is what still makes each family visually distinct.

3. **The CSS *rules* that consume those tokens convert to Tailwind
   utilities, compiled with arbitrary-value syntax that references the same
   custom properties**, e.g. a rule that was
   `.studio-card{background:var(--surface);border:1px solid var(--line);
   border-radius:var(--radius-lg)}`
   becomes a Tailwind source rule
   `.studio-card{@apply tw:border tw:rounded-[var(--radius-lg)];
   background-color:var(--surface);border-color:var(--line)}`
   compiled by the standalone CLI into real CSS. **The selector never
   changes** - `.studio-card` stays `.studio-card`. Properties Tailwind has
   no utility for (CSS custom properties, multi-value shorthand like
   `padding:16px 18px`, `font:600 .78rem var(--font-sans)`, exotic
   `:has()`/`color-mix()` expressions) stay as plain CSS declarations
   alongside the `@apply` line, in the same rule - this hybrid is deliberate
   and documented here as the accepted approach, not a gap: forcing every
   multi-value shorthand into an awkward combination of single-axis
   utilities would make the source harder to read for no benefit.

4. **Compiled output stays unlayered CSS** (no `@layer components` wrapper).
   This was verified directly: `@apply` resolves correctly whether or not
   the rule is inside a Tailwind layer, and emitting it *unlayered* is load
   bearing - CSS cascade layers always lose to unlayered CSS regardless of
   specificity or source order, so wrapping converted rules in
   `@layer components` while other, not-yet-converted consts remain
   ordinary CSS would have silently inverted priority between converted and
   unconverted rules. Keeping output unlayered preserves the exact same
   cascade behavior the hand-written CSS had, which is what makes an
   **incremental**, const-by-const conversion safe rather than an
   all-or-nothing atomic rewrite.

5. **The `tw:` prefix is still used** (ADR-0338/0339's pattern) even though,
   for the admin shell specifically, there is no risk of colliding with
   Tailwind's own default token names (`--radius-lg` etc. - the codebase
   uses its own ADR-0053 names already). The prefix is kept for consistency
   with the rest of the Tailwind tooling in this repo and because it costs
   nothing. Tailwind's **preflight** reset layer is excluded (not imported)
   for the same reason it was excluded for Aurora: it would reset element
   defaults unscoped, across the whole admin shell page, not just the
   converted selectors.

6. Build tooling is unchanged from ADR-0338: the standalone CLI is
   downloaded on demand, used to compile, then discarded (not vendored);
   the compiled CSS is pasted back into the same Go string constants admin
   Studio already uses (`AdminCSS`, `AdminShadcnCSS`, `AdminWorkbenchCSS`,
   etc.) - **no new Go files, no new `go:embed`, no change to how
   `admin_shell.go` concatenates and serves these constants.** This keeps
   the blast radius to "the value of some string constants changed," not
   "the page assembly pipeline changed."

## Alternatives

- **Fork the shared markup per family and hand-write Tailwind utility
  classes directly in each family's own HTML copy.** Rejected: this is
  what ADR-0339 did for Aurora *by design*, as a one-off exception, because
  Aurora deliberately diverges in structure. Doing the same for
  hydrogo/shadcn/workbench/nova would quadruple the markup surface,
  contradict AGENTS.md's "never fork for a specific need," and break the
  existing shared-markup contract tests outright instead of requiring the
  cosmetic text-format update this ADR's approach needed.
- **Wrap converted rules in `@layer components`.** Rejected after testing:
  CSS cascade layers always lose to unlayered CSS, so this would make any
  still-unconverted (plain CSS) rule automatically outrank a converted one
  regardless of specificity or source order - a real, non-obvious
  regression risk for a partial/incremental migration. Verified directly
  with the standalone CLI (`@apply` resolves identically whether layered or
  not) before choosing the unlayered approach.
- **Convert every declaration to a utility, including multi-value
  shorthand, by hand-splitting into single-axis utilities.** Rejected as
  needless churn: Tailwind's utility model does not have a shorthand
  arbitrary-value utility for e.g. `padding:16px 18px` or
  `font:600 .78rem var(--font-sans)`; forcing these into verbose
  multi-utility combinations buys nothing over keeping them as plain CSS
  declarations inside the same (now partially Tailwind) rule.
- **A single hand-rolled automated converter trusted blindly for the full
  ~84KB of admin CSS plus the ~4000-line public theme CSS in one pass.**
  Rejected for the public theme and for several large admin consts (see
  "Remaining work" below): there is no way to visually verify the live
  marketing site or the full admin surface in this environment (no
  screenshot/browser tool was available), so converting everything in one
  unreviewed pass would have traded a documented, bounded risk for an
  undocumented, unbounded one. The mechanical conversion approach was
  instead validated on a representative, fully-tested subset first.

## Consequences

### Benefits

- Admin Studio's component styling is authored in Tailwind for the
  highest-traffic shared base layer and all four families' component
  overrides (see Validation for the exact list converted in this change),
  with zero markup changes and zero change to which families exist or how
  they're selected.
- The shared-markup/token-swapping architecture (ADR-0041/ADR-0053) is
  fully intact and still enforced by its existing contract tests.
- The pattern (selector-preserving `@apply`, unlayered output, hybrid
  utility+raw-CSS per rule) is proven correct and is safe to continue
  incrementally for the remaining consts without needing a new ADR each
  time - this document is the standing authorization for that pattern.

### Costs and risks

- **This change is partially delivered, not a full "everything" migration.**
  See "Remaining work" - several large consts and the public theme are
  explicitly deferred. Treat this ADR as establishing the pattern and
  converting a representative, fully-verified majority of admin Studio's
  component CSS, not as a claim that no hand-written CSS remains anywhere.
- Attribute selectors in Tailwind's compiled output drop quotes where CSS
  allows it (`[data-ui-theme=shadcn]` instead of `[data-ui-theme="shadcn"]`)
  - functionally identical, but several existing tests asserted the quoted
  text literally and needed updating (done as part of this change; see
  Validation).
- `background:X` / `color:X` declarations convert to `background-color:X` /
  `color:X` via Tailwind's `bg-`/`text-` utilities. This is only safe
  because every token this codebase's `background`/`color` declarations
  reference resolves to a plain color (verified by grep across all
  converted consts - no gradients or images were found); it would not be
  safe to apply this mechanical rule to a `background` declaration that
  could resolve to an image or gradient.
- No visual/screenshot verification was possible in this environment for
  any of the converted surfaces. Correctness here rests on: (a) the
  selectors never changing, (b) a declaration-count sanity check per
  converted file (compiled output never had *fewer* top-level rules than
  the original, only more - from Tailwind's own `@property`/`@supports`
  plumbing), (c) targeted manual verification of several non-trivial cases
  (box-shadow with `color-mix`, `!important` propagation through derived
  custom properties, multi-selector grouped rules), and (d) the full
  existing automated test suite passing unmodified in intent (only
  quote-format assertions were adjusted, not the invariants they check).
  A human visual pass before shipping to production is still recommended.

## Remaining work (explicitly deferred, not done in this change)

Converted in this change (see Validation): `AdminCSS`, `AdminPolishCSS`,
`AdminResponsiveCSS`, `AdminUXCSS`, `AdminShadcnCSS`, `AdminWorkbenchCSS`,
`AdminWorkbenchDensityCSS`, `AdminWorkbenchTypeCSS`,
`AdminWorkbenchComponentsCSS`, `AdminNovaCSS`, `AdminNovaDensityCSS`,
`AdminNovaTypeCSS`, `AdminNovaComponentsCSS` - the shared base layer plus
all four families' component-override layers.

**Not converted, and should not be assumed done:**

- The remaining smaller CSS/JS constants in `internal/ui` not listed above
  (e.g. `AdminDashboardCSS`, `AdminFormLayoutCSS`, `AdminMediaCSS`,
  `AdminToastCSS`, and others) - lower-traffic surfaces, same pattern
  applies, not yet executed.
- `adminWorkbenchDashboardTemplate` and the Nova dashboard template's own
  markup were **not** rewritten with Tailwind utility classes in-line; only
  their *CSS* (`AdminWorkbenchComponentsCSS`, `AdminNovaComponentsCSS`,
  etc.) was converted. Converting the markup itself is a different,
  higher-risk change (editing the HTML template literals directly) that
  was out of scope for this pass.
- `themes/corporate/` (the public marketing site: a ~4000-line hand-written
  `theme.css` and five block templates) was **not touched**. This is a
  separate, filesystem-loaded theme system (`extension/v1`) from admin
  Studio, carries live-site risk, and is an order of magnitude larger than
  what this change converted. It should be scoped and executed as its own
  follow-up using the same validated pattern (selector-preserving `@apply`,
  unlayered compiled output), with a human visual review step this
  environment could not provide.

## Scale impact

- Compact: no change - build-time-only compilation, same embedded Go
  string constants as before, no new runtime dependency, no new service.
- Standard / Distributed: no change.

## Database impact

N/A - no schema, query, or storage change.

## Cache impact

N/A - no server-side cache key, TTL, or invalidation introduced; this is
static CSS baked into the binary at the same place it always was.

## Deployment and compatibility

Pure content change to existing Go string constants - no new routes,
migrations, or embedded files, no change to `admin_shell.go`'s structure or
concatenation order. Old and new binaries can coexist during a Blue/Green
rollout with no compatibility shim: the rendered page's class names and
structure are byte-identical, only the CSS text backing them differs.

## Validation

- `go build ./...` clean.
- `go test ./internal/ui/... ./internal/... ./modules/... ./cmd/...` all
  green (full run, not just the touched packages).
- Existing contract tests updated to match Tailwind's (functionally
  identical) attribute-selector quoting, not deleted or weakened:
  `TestAdminShellIncludesShadcnThemeModes`,
  `TestAdminShellIncludesWorkbenchThemeMode`,
  `TestAdminShellIncludesNovaThemeMode`,
  `TestHydroAndShadcnModesShareSemanticMarkupContract`,
  `TestNovaModeSharesSemanticMarkupContract`,
  `TestBrandBackgroundsNeverPairWithHardcodedWhiteText` (also updated to
  match `background-color:` replacing `background:` under the `bg-`
  utility, preserving the real regression guard it implements) -
  all in `internal/ui/admin_areas_test.go` and
  `internal/ui/admin_dashboard_kits_test.go`.
- Per-file sanity check performed during conversion (not committed as an
  automated test, since it required ephemeral tooling and the original
  hand-written CSS for comparison): for all 13 converted constants, a
  top-level-rule count comparison confirmed the compiled output never has
  *fewer* rules than the original (only more, from Tailwind's own
  `@property`/`@supports` plumbing), and every `@media` condition in the
  original was confirmed present in the compiled output.
- `scripts/theme-token-lint.ps1` is unaffected (it lints `themes/`, which
  this change does not touch).

## References

- `docs/architecture/0041-admin-visual-theme-modes.md`
- `docs/architecture/0053-semantic-ui-token-contract.md`
- `docs/architecture/0338-standalone-tailwind-build-time-css.md`
- `docs/architecture/0339-tailwind-admin-visual-family.md`
- `internal/ui/admin_shell.go`, `internal/ui/admin_workbench.go`,
  `internal/ui/admin_nova.go`, `internal/ui/admin_ux.go`
