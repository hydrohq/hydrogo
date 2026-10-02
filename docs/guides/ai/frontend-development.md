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
