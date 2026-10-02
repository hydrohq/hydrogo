# ADR-0340: Tailwind utilities as the admin Studio and public theme
  styling language

- **Status:** Accepted (admin Studio: the shared base layer and all four
  families' component CSS, plus the two dashboard templates' markup, are
  converted; the public `corporate` theme's `theme.css` is fully converted;
  its five block templates' markup is a deliberate, documented exception -
  see Validation and "Remaining work")
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

## Remaining work and deliberate exceptions

Converted across this change, in two passes:

**Admin Studio CSS** (all of it, not just a subset): `AdminCSS`,
`AdminPolishCSS`, `AdminResponsiveCSS`, `AdminUXCSS`, `AdminShadcnCSS`,
`AdminWorkbenchCSS`/`Density`/`Type`/`Components`,
`AdminNovaCSS`/`Density`/`Type`/`Components`, plus every remaining
lower-traffic constant (`AdminAreaCSS`, `AdminCSSOverrides`,
`AdminDashboardCSS`, `AdminFamilySelectCSS`, `AdminFormLayoutCSS`,
`AdminIconCSS`, `AdminInfoTipCSS`, `AdminItemListCSS`, `AdminKitCSS`,
`AdminListViewCSS`, `AdminMediaCSS`, `AdminMediaFormsCSS`,
`AdminNavTreeCSS`, `AdminNovaDashboardCSS`, `AdminNovaDashboardLayoutCSS`,
`AdminPasswordStrengthCSS`, `AdminPermissionsCSS`, `AdminResourceCSS`,
`AdminRevisionCSS`, `AdminRichTextCSS`, `AdminRichTextFixCSS`,
`AdminSearchCSS`, `AdminSplitEditorCSS`, `AdminSpotlightCSS`,
`AdminTabsCSS`, `AdminToastCSS`, `AdminWorkbenchDashboardCSS`).

**Admin dashboard markup**: `adminWorkbenchDashboardTemplate`'s and the
Nova dashboard template's own HTML now carry `tw:`-prefixed utility
classes directly (base/default state only - see the dedicated
`internal/ui/dashboardtw` package), alongside their original class names
kept as hooks. `:hover` states and `@media` responsive overrides for this
markup are a deliberate, narrower exception: they remain served by the
already-converted selector-based CSS rather than being re-expressed as
`hover:`/responsive Tailwind variants in the markup, since that CSS is
already correct and acts as a safety net.

**The public `corporate` theme's `theme.css`** (~4000 lines): fully
converted using the identical pattern proven on admin Studio - `@apply`
against the theme's own existing selectors in a new
`themes/corporate/static/theme.source.css`, compiled by the standalone CLI
(`scripts/build-theme-tailwind.ps1`) into `static/theme.css` with **zero
changes to the five block templates' markup or to the theme's visual
output**. This is a separate, filesystem-loaded theme system
(`extension/v1`), not part of admin Studio, and the compiled asset's
cache-buster hash was updated everywhere it's referenced (see Deployment
and compatibility).

**Not converted, and should not be assumed done:**

- The five block templates under `themes/corporate/blocks/*.html`
  (`cta.html`, `faq.html`, `hero.html`, `pricing_table.html`,
  `testimonials.html`) were **deliberately left with their existing
  semantic class names**, not given inline Tailwind utility classes the
  way the admin dashboard templates were. This is a considered exception,
  not an oversight: every class these templates use is already fully
  re-styled by the converted `theme.css`, so the rendered output is
  byte-for-byte the same either way - adding utility classes on top would
  be cosmetic-only duplication, not a functional change, and converting
  markup on a live, unscreenshotable production site carries exactly the
  kind of risk this ADR's methodology is built to avoid when it buys
  nothing. `hero.html` in particular (255 lines, the homepage's first
  visual element) was the deciding factor for treating all five uniformly
  rather than converting the four smaller ones and leaving only the
  riskiest one out.

### Bugs found and fixed while converting `theme.css` (read before extending this pattern)

Converting a CSS file an order of magnitude larger than anything touched
so far surfaced three real correctness bugs in the conversion *tooling*
(not in any previously-committed output - each was checked against the
already-committed admin CSS and confirmed absent there; see the specific
notes below). Anyone extending this pattern to new CSS must be aware of
these:

1. **A comment directly before `@media`/`@keyframes`/`@font-face` defeats
   the opaque-block detection.** The converter checks
   `strings.HasPrefix(raw, "@media")` etc. to decide a chunk must be passed
   through structurally instead of parsed as flat `prop:value` pairs; a
   leading `/* comment */` makes that prefix check fail, so the chunk falls
   through to the flat parser and its *nested* rules get corrupted (the
   same failure mode as #2, below, triggered a different way). Fix:
   strip all CSS comments before parsing, not after. Checked against every
   admin CSS constant converted earlier - none had a comment immediately
   before an `@media`/`@keyframes` block, so that already-committed output
   is unaffected; this was latent, not live.
2. **`@keyframes` bodies are not flat `prop:value` lists** - each
   percentage/from/to stop is its own nested rule. Without special-casing
   `@keyframes` as an opaque block (emitted verbatim, never parsed into
   declarations), the flat parser's semicolon-splitting can corrupt the
   stop structure. This two depended on #1 to manifest in several cases
   actually observed in `AdminSpotlightCSS` during the admin conversion
   pass; it was caught and fixed there already, and separately re-verified
   safe for `theme.css`'s two keyframe blocks here.
3. **Tailwind's arbitrary-value utilities are ambiguous for some property
   pairs, and the type inference can guess wrong for `var()` values and
   the keyword `none`:**
   - `font-weight` and `font-family` both resolve through Tailwind's single
     `font-[value]` utility (it infers which one from the value's shape).
     `font-family: var(--font-mono)` was incorrectly inferred as a
     **weight** (`font-weight:var(--font-mono)`), silently discarding the
     monospace font. Fix: map `font-weight` to a utility only when the
     value is a plain integer (`strconv.Atoi` succeeds); leave
     `font-family` as plain CSS always.
   - `background`, `background-color`, and `background-image` all resolve
     through `bg-[value]`. Gradients and `url()` are correctly inferred as
     images (verified directly, multiple cases) and plain colors (including
     `var()` references) are correctly inferred as colors (verified
     directly) - but the keyword **`none`** was inferred as a color,
     producing the invalid declaration `background-color:none` (silently
     dropped by browsers - harmless here only because `background-image`'s
     and `background-color`'s own initial values already mean "no image" /
     relies on a separate rule for the actual color, but not something to
     rely on in general). Fix: leave the value `none` as plain CSS for all
     three of these properties.
   - `transition` and `transform` were removed from the mapping entirely
     (never attempted, not even conditionally), because Tailwind's
     `transition-[value]`/`transform-[value]` arbitrary utilities don't
     treat the bracket content as an opaque passthrough: `transition-[...]`
     extracts a property list and separately injects Tailwind's *own*
     configured default `transition-duration`/`transition-timing-function`
     custom properties, ignoring the duration/easing written inside the
     brackets - confirmed directly (`transition: transform .16s
     cubic-bezier(...)` compiled to Tailwind's default ~150ms/
     `cubic-bezier(.4,0,.2,1)` timing, not the authored one). This is a
     real, user-visible behavior change (animation speed/easing), not a
     formatting difference, and unlike the other two it cannot be scoped
     down to specific values - any shorthand transition/transform with a
     component other than what Tailwind's own token scale expects is at
     risk. Both properties are **excluded from this conversion pattern
     categorically** going forward, not just for `theme.css`.
   None of these three affected the already-committed admin Studio CSS:
   grepped for `font-weight:var(--font`, `background-color:none`, and
   `transition-property`/`transform-property` signatures across every
   converted admin file before writing this section - zero matches.
   `internal/ui/dashboardtw`'s markup-level classes were hand-derived per
   element (not run through the buggy automated mapping) and were
   independently spot-checked against these same three patterns.

Given these findings, **extending this pattern to new CSS should route
`font-weight`, `background`/`background-image`, `transition`, and
`transform` through the same restricted handling** (numeric-only for
font-weight, never-"none" for background, always-raw for
transition/transform) rather than the naive "map every declaration with a
same-named utility" approach this ADR started with.

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

For admin Studio: pure content change to existing Go string constants - no
new routes, migrations, or embedded files, no change to `admin_shell.go`'s
structure or concatenation order (aside from the new, additive
`internal/ui/dashboardtw` package's `CSSText` being appended to the same
concatenation chain). Old and new binaries can coexist during a Blue/Green
rollout with no compatibility shim: the rendered page's class names and
structure are byte-identical, only the CSS text backing them differs.

For the public theme: `themes/corporate/static/theme.css`'s content
changed, so its cache-busting version string changed too
(`?v=e0fcce17` -> `?v=00013d55`, the new file's SHA-256 prefix, consistent
with how `theme.js`'s existing `?v=` is derived - see
`tools/assetmanifest`). Updated everywhere referenced:
`themes/corporate/layouts/base.html` and the two hardcoded admin-free page
renderers in `modules/members/transport/http.go` and
`modules/news/transport/http.go`. `tools/assetguard` (part of
`scripts/verify-local.ps1`) enforces that every reference uses the same
version string - it was run and passes. A deployed old binary serving the
old `theme.css` alongside a new binary serving the new one during a
Blue/Green rollout is safe either way: browsers cache by the full URL
including `?v=`, so old and new clients each get the asset matching the
page they loaded, never a mismatched pairing.

## Validation

- `go build ./...` clean.
- `go test ./internal/... ./modules/... ./cmd/... ./tests/... ./extension/...
  ./tools/...` all green (full run, not just the touched packages), across
  both the admin Studio pass and the public theme pass.
- Existing contract tests updated to match Tailwind's (functionally
  identical) attribute-selector quoting and declaration reordering, not
  deleted or weakened: `TestAdminShellIncludesShadcnThemeModes`,
  `TestAdminShellIncludesWorkbenchThemeMode`,
  `TestAdminShellIncludesNovaThemeMode`,
  `TestHydroAndShadcnModesShareSemanticMarkupContract`,
  `TestNovaModeSharesSemanticMarkupContract`,
  `TestBrandBackgroundsNeverPairWithHardcodedWhiteText`,
  `TestAdminShellIncludesNovaTopNav`,
  `TestAdminShellHidesInactiveDashboardVariantByCSS`,
  `TestMediaToolbarRowButtonsShareTheSameHeightUnderNova`,
  `TestNovaThemeToggleAndAvatarHaveVisibleBorders`,
  `TestNovaDashboardHidesPagesCollectionAndGatesCommerceAction` (prefix-match
  on `class="nova-X "` instead of the old exact `class="nova-X"`, since the
  attribute now also carries utility classes) - all in
  `internal/ui/admin_areas_test.go` and `internal/ui/admin_dashboard_kits_test.go`.
  A `firstRuleBody` test helper was added for order-insensitive declaration
  checks where Tailwind reordered properties within a rule.
- `internal/ui/dashboardtw`'s contract tests (mirroring Aurora's pattern):
  embedded CSS present, no preflight, no unprefixed semantic-token
  definition, and a sample of the 157 distinct `tw:` classes used in the
  two dashboard templates confirmed present in the compiled output (all 157
  were checked during development, not just the sample kept in the test).
- `extension/v1/theme_tailwind_test.go` (new): the corporate theme's
  compiled `theme.css` is non-empty, excludes Tailwind's preflight,
  contains real selectors for a representative sample of classes the block
  templates render, and specifically regression-guards the two `bg-`/`font-`
  ambiguity bugs found during conversion (asserts `font-family:var(--font-sans)`
  is present and `font-weight:var(--font-sans)` / `background-color:none`
  are absent).
- Declaration-level parity verification (not committed as an automated
  test - it required ephemeral tooling and the pre-conversion original for
  comparison, which only exists in git history once this lands): for every
  converted file (all converted admin CSS constants and `theme.css`), a
  tool parsed both the original and compiled CSS into
  `(media, selector) -> declared properties` and confirmed no property was
  lost, accounting for Tailwind's selector grouping (identical-body rules
  merged with commas), legacy pseudo-element normalization (`::before` ->
  `:before`), and combinator whitespace normalization. For `theme.css`
  specifically, every one of the ~40 remaining reported diffs after those
  adjustments was manually traced to a confirmed-safe cause (redundant
  default-value dropping such as `backdrop-filter:none` or the `rotate(0deg)`
  keyframe stop, or Lightning CSS's border-shorthand-plus-override merge) -
  none were left unexplained.
- `go run ./tools/assetguard` and `scripts/theme-token-lint.ps1` both pass
  against the new `theme.css`.

## References

- `docs/architecture/0041-admin-visual-theme-modes.md`
- `docs/architecture/0053-semantic-ui-token-contract.md`
- `docs/architecture/0338-standalone-tailwind-build-time-css.md`
- `docs/architecture/0339-tailwind-admin-visual-family.md`
- `internal/ui/admin_shell.go`, `internal/ui/admin_workbench.go`,
  `internal/ui/admin_nova.go`, `internal/ui/admin_nova_dashboard.go`,
  `internal/ui/admin_ux.go`
- `internal/ui/dashboardtw/` (admin dashboard markup's compiled CSS),
  `scripts/build-dashboard-tailwind.ps1`
- `themes/corporate/static/theme.source.css` (Tailwind source),
  `themes/corporate/static/theme.css` (compiled output),
  `scripts/build-theme-tailwind.ps1`, `extension/v1/theme_tailwind_test.go`
