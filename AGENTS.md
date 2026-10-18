# Working in HydroGo

HydroGo is a **single-binary headless Go CMS**: pages, news, media, members,
content types, a REST API and an embedded admin panel. It does not render a
public website; projects build their own frontend on the API (see
`examples/svelte-starter`). It is not a framework or a product factory. If a
change does not serve that, don't make it. Background:
`docs/decisions/0001-cms-focus.md`, `docs/decisions/0005-admin-svelte-spa.md`.

## Before you finish any task

Run `go run ./tools/check` (or `make check`; build, vet, gofmt, guards, Go
tests, admin lint/typecheck/tests/build). If you changed admin UI, also open
the screen and look at it (see "Admin UI work" below). Don't report a task as
done if either step failed.

## Where things go

| You are adding… | Put it in |
|---|---|
| A simple content type (FAQ, team, events, …) | one YAML file in `content-types/` — no Go, no Svelte. The admin screens and `/api/v1/content/<id>` come from the schema. See `content-types/faq.yaml` and `internal/contenttypes/definition.go` for the field kinds |
| Content module logic (pages, news, members, site) | `modules/<name>/` → `model.go`, `service/` or `service.go`, `repository/sqlstore/`, `transport/http.go` (JSON only) |
| Shared core capability | `internal/<name>/` |
| An admin screen for a module | `admin/src/routes/<module>/` (Svelte), built from `admin/src/lib/components` (shadcn-svelte) and the schema-driven pieces in `admin/src/lib/schema` |
| An admin API endpoint | `/api/v1/admin/...` in the module's `transport/http.go`, responses through `internal/api.WriteData` / `WriteError` |
| Database change | new directory in `migrations/` with `migration.yaml`, `sqlite.sql`, `postgres.sql`, `mysql.sql` |
| CLI command | its own file in `cmd/hydro/<name>_cmd.go`, one `case` line in `run()` |
| Example of consuming the API from a frontend | `examples/svelte-starter/`. `examples/hydropost-news-starter/` is another one (a Turkish news portal) — read its own `AGENTS.md` before touching it: its design must never be copied or imitated for other frontend work, each one gets designed from scratch |

Follow the closest existing example. `modules/news` is the reference module
for content that needs custom behaviour; everything else should be a content
type. Content types get admin screens at `/admin/content/<id>` and, with
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
9. **Regenerate, don't hand-edit**, generated files: `admin/src/lib/api/types.ts`
   comes from `make generate` (`hydro schema openapi` → `openapi-typescript`);
   `admin/dist` comes from `vite build`.
10. **Every API response uses the envelope** in `internal/api/response.go`:
    `{data, meta}` or `{code, message, request_id, fields}`. Errors must carry
    the request id so the admin can show it.
11. **Every admin endpoint works with all three credentials** — session
    cookie + CSRF header, `Authorization: Bearer`, or API key — through the
    shared auth middleware. Don't add per-handler auth parsing.
12. **Zero build on content publish.** Publishing, editing, or retracting a page,
    news article, or content record must never require a frontend rebuild step,
    static pre-rendering pipeline, or CI/CD re-deployment. Content changes must be
    live immediately via the API and database.

## Admin UI work

- Stack: Svelte 5 (runes) + TypeScript + Tailwind + shadcn-svelte, under
  `admin/`. Data access goes through `admin/src/lib/api/client.ts` (CSRF
  header, 401 handling, envelope parsing) and `@tanstack/svelte-query`.
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
go run ./tools/check       # everything CI checks that runs fast locally (or: make check)
make generate              # OpenAPI → admin/src/lib/api/types.ts
go test ./modules/news/... # one Go package
npm --prefix admin test    # admin unit tests
```
