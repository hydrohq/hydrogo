# Frontend development (AI guide)

"Frontend" in this repo means **two separate systems** — don't conflate
them, they use different languages, build steps, and live in different
directories.

| | Public site theme | Admin Studio |
|---|---|---|
| Audience | Site visitors | Staff/editors |
| Location | `themes/<name>/` | `modules/designsystem/`, `internal/ui/` |
| Markup language | Plain HTML (`.html` files) | Templ (`.templ` → generated `_templ.go`) |
| Styling | Tailwind, compiled per-theme | Tailwind, compiled per admin package |
| Rebuild trigger | Manual script run | Automatic under `hydro run` |

## Public theme (`themes/<name>/`)

Structure (e.g. `themes/corporate/`):
```
theme.json              # name, version, design tokens, slots, asset list
layouts/base.html        # the page shell
blocks/*.html            # hero.html, cta.html, faq.html, pricing_table.html, testimonials.html
static/theme.source.css  # Tailwind-authored source — edit THIS
static/theme.css         # compiled output — generated, don't hand-edit
static/theme.js
```

Edit `blocks/*.html` or `layouts/base.html` directly for markup changes —
these are plain HTML, no build step needed for them specifically.

### Changing theme CSS

`static/theme.css` is compiled from `static/theme.source.css`. Edit the
source, then:
```powershell
scripts/build-theme-tailwind.ps1 -Theme <name>
```
(Needs the standalone `tailwindcss` CLI — download the matching binary from
`https://github.com/tailwindlabs/tailwindcss/releases`, asset named
`tailwindcss-<os>-<arch>[.exe]`, no Node/npm involved. Pass `-TailwindCli
<path>` if it's not on `PATH`.) This script does **not** import Tailwind's
preflight reset — the theme's own `theme.source.css` already defines a
complete top-level reset; importing preflight on top would silently change
element defaults the theme doesn't expect. Don't add a preflight import
without understanding this.

**After rebuilding, you must manually update a cache-busting hash** — the
script's own output tells you this, don't skip it:
```sh
go run ./tools/assetmanifest -root themes
```
Take the new file's first 8 hex chars of its SHA-256 and update the
`theme.css?v=<hash>` query string in all three places:
- `themes/<name>/layouts/base.html`
- `modules/members/transport/http.go`
- `modules/news/transport/http.go`

Forgetting this doesn't break anything locally but fails
`scripts/verify-local.ps1`'s `assetguard` step (mismatched versions) — the
same check that runs on `git push` per this repo's pre-push hook. If a push
fails with an assetguard error, this is almost certainly why.

There's also a token lint (`scripts/theme-token-lint.ps1`, part of
`verify-local.ps1`): any `var(--some-token)` reference in a theme CSS file
must have a matching `--some-token: ...` definition somewhere under
`themes/`, unless it has a local fallback (`var(--token, fallback)`, which
is exempt). A CSS edit that references an undefined custom property fails
this check.

## Admin Studio (`modules/designsystem/`, `internal/ui/`)

Components are written in **Templ** (`.templ` files, compiled to
`_templ.go` by the `templ` CLI). **Never hand-edit a `_templ.go` file** —
it's regenerated and your change is silently lost.

Under `hydro run` (see `docs/guides/ai/local-development-setup.md`), Templ
files are **auto-watched and regenerated** — just edit the `.templ` file
and save; no manual `templ generate` needed while the dev server is
running. Outside that (e.g. CI, a one-off check), run it explicitly:
```sh
templ generate
```

Design System packages that compile their own Tailwind CSS (check
`scripts/build-*-tailwind.ps1` for the current list — `aurora`, `dashboard`,
`hydrogoui`, `shadcnui`, plus the general `build-tailwind.ps1` and
`build-docs-tailwind.ps1` for the docs portal) follow the exact same
source→compiled→cache-bust pattern as the theme CSS above. Check the
specific script for the package you're touching rather than assuming they
all work identically — some target a single package, some are shared.

Static JS assets for specific widgets (not full page scripts) live in
`modules/designsystem/assets/*.js` — e.g. `feedback.js`, `article-reading.js`
were added specifically so a widget's interactive behavior could be
decoupled from server-rendered, CSP-nonce'd inline `<script>` blocks (see
`docs/architecture/0265-fastcgi-and-persistent-html-cache.md` and the
`cache-everywhere` work in git log for why: an inline nonce'd script makes
its page uncacheable). If you're adding a new widget with any client-side
behavior, prefer an external asset file over an inline `<script nonce>`
block for exactly this reason, unless the page it lives on is never going
to be anonymous/cacheable anyway (e.g. anything under `/admin/`).

## Verifying a frontend change

- `go build ./...` — catches any Go-side template binding break.
- `scripts/verify-local.ps1` (or at minimum its `assetguard`, theme token
  lint, and HTML-comment-lint steps) before pushing — these are the exact
  checks the pre-push hook runs; failing them blocks `git push`.
- For a public theme change, actually load the affected public route
  (`hydro run`, `http://127.0.0.1:8090/...`) and look at it — a lint pass
  doesn't prove the page looks right.

## Producing high-quality UI with Templ and Tailwind

Target the same visual quality as a well-designed Next.js/React frontend.
HTML/CSS rendering supports the same typography, layout, responsive behavior,
and visual effects. Use HydroGo's existing components and design contracts to
make that quality repeatable. Do not assume that switching frameworks alone
improves the design.

### Establish the design brief

1. Identify the audience, primary task, actual content, supported languages,
   and the surface being changed: public theme, Admin Studio, or Edition UI.
2. Read [DESIGN.md](../../../DESIGN.md), the affected theme's sources, and
   relevant ADRs. Apply explicit user preferences for the requested surface;
   do not propagate one surface's palette to unrelated themes.
3. Establish reference designs and a consistent type scale, content measure,
   spacing scale, color roles, radii, and shadows. Reuse the
   [semantic token contract](../../architecture/0053-semantic-ui-token-contract.md).
   Avoid inventing a separate token vocabulary for each page.

### Compose from existing components

- Inventory `internal/ui/components/` and `modules/designsystem/` before
  adding primitives. Reuse typed Button, Input, Card, and Form contracts and
  existing blocks; keep escaping and accessibility at the component boundary.
- For dialogs, menus, tabs, tables, notices, and empty states, find an existing
  implementation first. Add a missing capability at the appropriate component
  boundary rather than copying markup between pages. Preserve flexibility in
  page composition instead of making every page use the same card layout.
- Put customer-specific UI and behavior under the Client Edition's `custom/`
  ownership. Change the blueprint/generator for machine-owned output. Never
  hand-edit `generated/` or `_templ.go` files.
- Use the affected surface's Tailwind entry and build script described above.
  Preserve shared semantic selectors and token definitions in Admin Studio;
  do not load isolated Edition Tailwind output into an existing admin family.
- A new component contract or HTMX capability requires the repository's ADR
  and verification process. Do not treat a styling request as authorization
  to invent routes, authorization policy, workers, or module activation.

### Implement behavior and complete states

Cover default, focus, disabled, loading, empty, error, and success states as
applicable. Use real content, long labels, and both English and Turkish when
the surface supports them. Provide meaningful labels, visible keyboard
focus, sufficient contrast, and keyboard access to all actions.

Use native controls, CSS, and small scoped JavaScript assets for client-side
interactions. Follow the external-asset/cache guidance above. Use HTMX only
through the registered allowlist and existing contracts; every mutation must
declare target, swap, CSRF, authorization, tenant scope, and idempotency.
Respect `prefers-reduced-motion` and keep inactive-module assets absent.

### Iterate against rendered evidence

1. Open the real affected route after rebuilding the required assets.
2. Inspect representative widths of 360, 768, and 1440 pixels. Adjust them
   when the supported device contract requires additional sizes.
3. Check content order, line length, alignment, clipping, horizontal overflow,
   and control usability. Exercise keyboard navigation and reduced motion.
4. Check real, empty, and long-content states, supported locales, and any
   supported light/dark modes. A component screenshot alone does not verify
   its page context.
5. Capture and inspect screenshots. Fix concrete defects, then recheck the
   changed states. Build/lint success is not visual acceptance.
6. Complete relevant asset/token checks and mandatory moduleguard,
   migrationguard, and importguard gates. Report actual results and existing
   failures; do not claim a failed gate passed.

For AI-assisted implementation, require a brief containing references,
existing tokens/components, actual content, interaction states, target widths,
and acceptance criteria. Replace vague requests such as "make it premium"
with those inputs, and use rendered evidence for subsequent corrections.

### Proposed improvements and framework choice

A component gallery showing variants, states, responsive behavior, and usage
examples, followed by a small set of complete reference screens (login,
admin list, editor form, content detail, landing page), is a recommended next
step. These are recommendations, not a claim that a unified gallery or full
reference-screen set is implemented. Verify current availability before
documenting or using one.

For a highly stateful visual editor or canvas application, assess a separate
Edition-owned React/Next.js frontend against the actual API contracts. Keep
its runtime and assets outside Core and the Compact baseline. Do not assume
an official starter, SDK coverage, or validated integration exists:
[ADR-0020](../../architecture/0020-headless-first-admin-studio-and-frontend-templates.md)
is **Proposed**, not an accepted implementation guarantee. Architecture and
contract changes still require their own ADR and evidence.

References:
[typed server UI](../../architecture/0209-typed-server-ui-contract.md),
[design-system provider boundary](../../architecture/0023-provider-agnostic-design-systems.md),
[blocks and lightweight interactions](../../architecture/0266-design-system-blocks-and-interactions.md),
[Tailwind authoring rules](../../architecture/0340-tailwind-styling-migration.md).
