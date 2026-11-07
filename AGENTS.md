# Working in HydroGo

HydroGo is a **single-binary headless Go CMS**: pages, news, media, members,
content types, a REST API and an embedded admin panel. It does not render a
public website; projects build their own frontend on the API (see
`examples/sveltekit-starter`). It is not a framework or a product factory. If a
change does not serve that, don't make it. Background:
`docs/decisions/0001-cms-focus.md`, `docs/decisions/0005-admin-svelte-spa.md`.

## Branches

`main` is the only long-lived branch (tags `vX.Y.Z` are cut from it). Work on a
short-lived branch and open a pull request into `main`. CI (`.github/workflows/`)
runs on every push/PR (Actions was re-enabled 2026-10-27). Still also run
`go run ./tools/check` yourself before finishing a task — don't wait on CI as
your only signal. Delete the branch once merged.

## Before you finish any task

Run `go run ./tools/check` (or `make check`; build, vet, gofmt, guards, Go
tests, a smoke test of the real server — security promises and speed on 10,000
records, `tests/smoke` — admin lint/typecheck/tests/build, and the starters'
check/test/build). Touching admin sign-in, users, invitations or a starter?
Add `-e2e`: real-browser journeys in `e2e/journeys` that start their own server. If you changed admin UI, also open
the screen and look at it (see "Admin UI work" below). Don't report a task as
done if either step failed.

## Where things go

| You are adding… | Put it in |
|---|---|
| A simple content type (FAQ, team, events, …) | one YAML file in `content-types/` — no Go, no Svelte. The admin screens and `/api/v1/content/<id>` come from the schema. See `content-types/faq.yaml` and `internal/contenttypes/definition.go` for the field kinds |
| Content module logic (pages, news, members, site, commerce) | `modules/<name>/` → `model.go`, `service/` or `service.go`, `repository/sqlstore/`, `transport/http.go` (JSON only) |
| Shared core capability | `internal/<name>/` |
| An admin screen for a module | `admin/src/routes/<module>/` (Svelte), built from `admin/src/lib/components` (shadcn-svelte) and the schema-driven pieces in `admin/src/lib/schema` |
| An admin API endpoint | `/api/v1/admin/...` in the module's `transport/http.go`, responses through `internal/api.WriteData` / `WriteError` |
| Database change | new directory in `migrations/` with `migration.yaml`, `sqlite.sql`, `postgres.sql`, `mysql.sql`. A new table most other tables have (check a sibling in the same module) needs a `tenant_id` column too — add it to `internal/tenant.BackfillTables`, `tests/contract/postgres_rls_contract_test.go`'s `expectedRLSTables`, and thread `tenant.Require`/`RequireTenantScope` through its repository in the same change. See `docs/multi-tenancy.md` and `docs/decisions/0024-news-taxonomy-tenant-isolation.md` for what happens when this is skipped |
| CLI command | its own file in `cmd/hydro/<name>_cmd.go`, one `case` line in `run()` |
| A starting point for a public site | `examples/sveltekit-starter/` (server-rendered, uses `packages/client`). `examples/crm-starter/` and `examples/commerce-starter/` are the `--starter crm`/`--starter commerce` panels for those presets. `examples/svelte-starter/` is HydroGo's own landing/docs SPA. `examples/hydropost-news-starter/` is a Turkish news portal — read its own `AGENTS.md` before touching it: its design must never be copied or imitated for other frontend work, each one gets designed from scratch |
| The project's own marketing/playground site (`hydro.ardalabs.dev`) | `examples/hydro-website/` — a static Svelte SPA with **no backend**: the playground view mirrors the real admin's shadcn/Zinc tokens and component markup (see its `src/lib/demo/`) against an in-memory, per-visitor fake data store, not a live HydroGo instance |
| The typed API client for frontends | `packages/client` (`@hydrogo/client`, private: never published to npm or any public registry). Keep it in step with the public API envelope: pages, news and content return `{data, metadata}` |
| What `hydro new` / `hydro init` create (presets, config template) | `internal/scaffold`. Presets only pick modules and stock types from `content-types/` |
| Release, installer, install scripts | `.github/workflows/release.yml`, `deploy/windows/hydrogo.iss`, `scripts/install.sh`, `scripts/install.ps1` (see `docs/upgrading.md`) |

Follow the closest existing example. `modules/news` is the reference module
for content that needs custom behaviour; everything else should be a content
type. `modules/seo` is an intentional exception to the module layout: it has
no persistence (llms.txt and OpenGraph generation only), so it has no
`model.go`, `repository/` or `transport/` — don't "fix" it into conformance. Content types get admin screens at `/admin/content/<id>` and, with
`public: true`, a read API at `/api/v1/content/<id>`; YAML flags add list
search/filters, drafts, revisions and per-locale (`localized`) fields (see
`definition.go`).

## Rules

1. **Go serves JSON, not markup.** No HTML or CSS in Go strings; no server-side
   HTML rendering except transactional e-mail (`internal/mailer`,
   `html/template`). Admin markup lives in `admin/` Svelte components.
2. **No customer-specific content in core.** Brand names, hardcoded menus,
   dates, exchange rates or demo credentials belong in seed data, config or
   the example frontend, not in `modules/`, `internal/` or `admin/`.
3. **Every migration ships all three dialects** (SQLite, PostgreSQL, MySQL)
   and never edits an already-released migration. Never delete a migration
   directory: databases that applied it refuse to start without it.
4. **No SQL dialect or cache details in services.** Keep them in
   `repository/sqlstore` and `internal/cache`.
5. **An inactive module adds nothing:** no routes, migrations, workers,
   assets or admin navigation entries.
6. **Keep files small.** Split a file before it passes about 800 lines
   (Go and Svelte). `cmd/hydro/main.go` is legacy; don't add to it, move code
   out of it.
7. **Don't write status reports, evidence logs or plans into the repo.**
   Put the explanation in the commit message. Add a decision record in
   `docs/decisions/` only when you change a module boundary or a public
   contract (routes, API, config, migrations).
8. **Never hide a failing test** by renaming it (`skip_TestX`), adding a
   bare `t.Skip` or `test.skip`. Fix the code, update the test to the new
   intended behaviour, or delete it with an explanation in the commit.
   `codeguard` rejects renamed tests.
9. **Don't hand-edit `admin/dist`**: it comes from `vite build`.
   `admin/src/lib/api/types.ts` is hand-maintained (there is no OpenAPI →
   TypeScript generation step yet); keep it in step with the admin API by
   hand when you change a response shape.
10. **Every API response uses the envelope** in `internal/api/response.go`:
    `{data, metadata}` or `{code, message, request_id, fields}`. Errors must carry
    the request id so the admin can show it.
11. **Every admin endpoint works with all three credentials** — session
    cookie + CSRF header, `Authorization: Bearer`, or API key — through the
    shared auth middleware. Don't add per-handler auth parsing.
12. **Zero build on content publish.** Publishing, editing, or retracting a page,
    news article, or content record must never require a frontend rebuild step,
    static pre-rendering pipeline, or CI/CD re-deployment. Content changes must be
    live immediately via the API and database.
13. **Unslop Design Standard: Zero emojis in UI buttons, badges, headers, or chrome.**
    Never use emojis (✨, 🚀, 👤, 🛒, 🔒, 💬, etc.) in buttons, action pills, navigation,
    or headings. Use clean typography or monochromatic SVG line icons (14–16px, 1.5–2px stroke).
    Keep interfaces alive through editorial rhythm, typography contrast, reading indicators,
    and micro-interactions rather than cheap visual noise.

## Admin UI work

- Stack: Svelte 5 (runes) + TypeScript + Tailwind + shadcn-svelte, under
  `admin/`. Data access goes through `admin/src/lib/api/client.ts` (CSRF
  header, 401 handling, envelope parsing) and `@tanstack/svelte-query`.
- Component tests use `@testing-library/svelte` + jsdom: add
  `// @vitest-environment jsdom` at the top of the test file (the suite's
  default environment is Node, since most `admin/src/lib` tests are plain
  logic). See `admin/src/lib/components/ui/ConfirmDialog.test.ts` and
  `Modal.test.ts` for the pattern, or `Header.test.ts` / `Sidebar.test.ts` /
  `DataTable.test.ts` for one that mocks `$lib/api/client` and sets the real
  `schema`/`auth`/`sidebar`/`theme` store state directly instead of a live
  endpoint or DOM measurement. `RichTextEditor.test.ts` mounts a real TipTap
  `Editor` in jsdom rather than faking it. `vitest-setup.ts` stubs
  `window.matchMedia`, which jsdom doesn't implement but the `theme` store
  calls on construction.
  Every `lib/components/` file now has a test except
  `shell/DashboardSkeleton` (pure decorative markup, not worth one). The 26
  `routes/` views are still untested at this level — add a component test
  alongside any of those you touch next, rather than leaving it for a
  dedicated sweep.
- Prefer the schema-driven screens: if a module's data fits
  `adminstudio.Collection`, let the generic list/edit components render it
  and add only the module-specific panels.
- Permissions: hide what `GET /api/v1/admin/me` doesn't allow, but the
  server is the authority.
- Show errors with their `request_id`; never swallow a failed request.
- Look at the result. Start the dev server (`go run ./tools/devserver`; it
  runs Vite with HMR for `admin/`), then screenshot the screens you changed at
  desktop and phone width and open the PNGs:

  ```bash
  HYDRO_ADMIN_EMAIL=... HYDRO_ADMIN_PASSWORD=... node e2e/screenshot.mjs /admin/content/faq
  ```

  It exits non-zero on HTTP errors or JavaScript errors. On Windows Git Bash,
  prefix it with `MSYS_NO_PATHCONV=1`.

## Commands

```bash
go run ./tools/devserver   # Go rebuilds + Vite HMR + demo data, http://127.0.0.1:8090
go run ./tools/check       # everything configured as CI checks -- run it yourself, Actions is disabled (or: make check)
go run ./tools/check -e2e   # also the browser journeys (needs `npx playwright install chromium`)
SMOKE_DIALECT=postgres HYDRO_TEST_POSTGRES_URL=... go test -tags smoke ./tests/smoke   # the same smoke tests on PostgreSQL (or mysql + HYDRO_TEST_MYSQL_URL)
go run ./tools/check -audit # also scan Go/npm dependencies for known vulnerabilities (needs network; configured to run weekly in CI, but see above)
go test ./modules/news/... # one Go package
npm --prefix admin test    # admin unit tests
```
