# 0005: Headless core, admin panel rebuilt as a Svelte SPA

- Status: Accepted
- Date: 2026-10-15
- Supersedes: 0004 (theme appearance CSS — the theme layer is removed)

## Context

The admin panel was Go + templ + htmx and the public site was rendered by
Go `html/template` themes (`themes/newsroom`). Two problems kept coming up:

- Design quality of templ/htmx and `html/template` screens lagged well
  behind what AI-assisted design tooling produces for JS component
  frameworks — the gap is in the tooling ecosystem, not in templ's
  correctness.
- Go, templ and htmx were three authoring models for one admin surface, and
  the theme layer was a fourth (`html/template` + theme CSS contract). The
  project's stated value is simplicity.

We measured before deciding:

- Go benchmark of the admin list render (`internal/adminstudio`, 20-row
  FAQ-shaped list) vs `json.Marshal` of the same rows: templ 29.6µs/op,
  22.7KB, 610 allocs; JSON 20.4µs/op, 8.0KB, 243 allocs. Serving JSON for a
  client-rendered admin is cheaper for the server than rendering HTML.
- Real HTTP load test (dev server, 20 published FAQ records, 2000 requests,
  concurrency 20): authenticated admin HTML page `/admin/resources/faq`
  967.6 req/s, 20.55ms avg, 36.6KB avg response; equivalent JSON
  `/api/v1/content/faq` 1081.3 req/s, 18.44ms avg, 8.6KB. Part of the gap
  is the admin's session check and part is the admin shell (nav, sidebar)
  repeated in every HTML response — an SPA loads the shell once.
- A spike rendering Svelte 5 server-side through a pure-Go JS engine (goja)
  works but requires patching Svelte's own runtime (it `import()`s
  `node:async_hooks`, which goja's parser rejects) and costs ~174µs/render
  vs ~4.5µs for `html/template`. Not needed: the admin has no SEO
  requirement, and the public site is no longer rendered by core.

## Decision

1. **The core is headless.** HydroGo serves a REST API and an embedded admin
   panel. It does not render public HTML. `themes/`, the theme contract
   (`extension/v1` theme views, `ThemeSiteView`, `AppearanceCSS`), public
   page/news/member HTML routes, the design-system block renderer
   (`modules/designsystem`), fonts, custom-code injection, error pages,
   contact widget, link check and the 404 redirect fallback are removed.
   Redirect rules, navigation, site settings, SEO settings, sitemaps and the
   OG image endpoint stay as data/API for whatever frontend a project builds.
   A SvelteKit example in `examples/svelte-starter` shows how to consume the
   API.
2. **The admin panel is a Svelte 5 SPA** (`admin/`: TypeScript, Tailwind,
   shadcn-svelte) built by Vite and embedded in the binary via `embed.FS`
   (`internal/adminui`). templ, htmx and the `internal/ui` component
   library are removed. Node is a build-time dependency only; the shipped
   artifact stays one `CGO_ENABLED=0` Go binary. A `no_ui` build tag
   produces an API-only binary.
3. **Admin screens are schema-driven.** `GET /api/v1/admin/schema` serves
   the `adminstudio.Collection` registry built from `content-types/*.yaml`;
   the SPA renders list/edit/new/drafts/revisions/locale screens from it.
   Adding a content type is still one YAML file and no code. Modules with
   custom behaviour (pages, news, members, site) expose JSON under
   `/api/v1/admin/<module>/...` and ship their own Svelte screens.
4. **One API, three credentials.** Same-origin SPA: `hydro_session` cookie +
   `hydro_csrf` double-submit header. Mobile/native/third-party clients:
   `Authorization: Bearer` tokens (the existing `/api/v1/mobile` bearer path
   generalised; staff personal access tokens). Server-side headless
   frontends: API keys with read/preview scopes. Public reads at
   `/api/v1/content/*` keep the allowlisted CORS middleware.
5. **Migration is delete-first.** The old admin and theme layer are removed
   in one change; the API and CLI keep working, and the SPA is rebuilt from
   scratch. There is no period where both admins coexist.
6. Admin extension surfaces (`/admin/extensions/*`, `extension/v1` signed
   HTML surfaces) are out of scope for v2 and removed with the templ admin.

Database support (SQLite, PostgreSQL, MySQL), `modules/*/repository/sqlstore`
and `migrations/` are untouched: this is a UI-layer change.

## Consequences

- `github.com/a-h/templ` leaves `go.mod`; `templ generate` leaves the dev
  server and `tools/check`. `tools/check` gains the frontend steps (`npm ci`,
  lint, `svelte-check`, vitest, `vite build`) and an OpenAPI freshness check;
  `admin/src/lib/api/types.ts` is generated from `hydro schema openapi`.
- Every admin HTML handler becomes a JSON handler using
  `internal/api.WriteData/WriteError`; the error envelope carries
  `request_id` end-to-end so the SPA can show it next to any failure.
- Admin-facing Playwright specs (`e2e/custom/admin-*`, `content-types`,
  `journeys`, `generated-richtext`) and the visual snapshot baseline are
  rewritten against the SPA. The Playwright/axe-core infrastructure
  (`playwright.config.ts`, `screenshot.mjs`, auth setup) is reused.
- Public-frontend behaviour is covered by API contract tests, not markup
  tests, because core no longer owns public markup.
- AGENTS.md is rewritten to describe the new layout and rules.
