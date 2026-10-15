# 0004: Theme contract carries the admin's Appearance CSS

- Status: Superseded by 0005 (the theme layer is removed; core is headless)
- Date: 2026-10-15

## Context

`modules/site`'s Appearance settings (admin-editable brand colors, dark
variants and font, WCAG-contrast-checked, `modules/site/appearance.go`)
already compiled to a `:root{--brand:...}` rule and were already injected
into the page `<head>` — but only for pages rendered through
`modules/pages`. `modules/news`, which renders the theme's actual homepage
(`/` redirects to it), never received it, so an admin-configured brand color
never reached most of the public site. `themes/newsroom`'s own CSS also
didn't consume any `--brand` variable; it hardcoded its accent color.

A second, unrelated token mechanism already existed on paper:
`extension/v1.ThemeManifest.Tokens` (`theme.json`'s `tokens` field). It is
validated but never compiled to CSS or rendered anywhere. It stays unused —
building a second wiring path for it would create two competing theming
systems. Appearance (DB-backed, admin-editable, contrast-checked) is the one
real token system.

## Decision

`extension/v1.ThemeSiteView` gains an `AppearanceCSS template.CSS` field.
Every module that renders a theme view populates it from the same
`site.AppearanceFromSettings(settings).CSS()` call (`modules/pages` already
did this through its own head-rendering path; `modules/news` and
`modules/members` now do too, via a `PublicContactSettings` handler field
and `modules/site` added to each module's declared dependencies in
`modules/dependencies.json`). Themes print it as
`<style>{{.AppearanceCSS}}</style>` in `<head>` when non-empty.

Because strict deployments run without `'unsafe-inline'` in `style-src`,
every module that injects this tag must also extend the response's CSP with
the exact SHA-256 hash of what it rendered. `modules/news/transport/public_head.go`
and `modules/members/transport/public_head.go` mirror `modules/pages/transport`'s
existing `allowHeadStyles`, built with `golang.org/x/net/html` (not a Go
string literal) so `codeguard`'s "no HTML in Go strings" rule still holds.

`themes/newsroom/static/theme.source.css` now sources its accent color from
`var(--brand, #dc2626)` instead of a hardcoded hex, so the shipped theme
re-skins automatically when Appearance changes, with no core or theme-file
edit required per site.

## Consequences

- Additive, zero-valued field; existing theme views and third-party themes
  that ignore it are unaffected.
- All three modules that render a public `ThemeSiteView` (`pages`, `news`,
  `members`) now populate `AppearanceCSS` the same way. Any future module
  added to that list must do the same (`allowAppearanceStyle` + its own
  `site.Setting` lookup declared in `modules/dependencies.json`) to stay
  consistent.
- `ThemeManifest.Tokens` (`theme.json`) stays declared-but-unused. Don't wire
  it up as a second branding mechanism; extend Appearance instead if more
  brand tokens are needed.
