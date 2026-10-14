# Working in HydroGo

HydroGo is a **single-binary Go CMS**: pages, news, media, members, themes, an
admin panel and a REST API. It is not a framework or a product factory. If a
change does not serve that, don't make it. Background:
`docs/decisions/0001-cms-focus.md`.

## Before you finish any task

Run `go run ./tools/check` (or `make check`; build, vet, gofmt, guards, tests). If you changed UI, also
open the page and look at it (see "UI work" below). Don't report a task as
done if either step failed.

## Where things go

| You are adding… | Put it in |
|---|---|
| A simple content type (FAQ, team, events, …) | one YAML file in `content-types/` — no Go code. See `content-types/faq.yaml` and `internal/contenttypes/definition.go` for the field kinds |
| Content module logic (pages, news, members, site) | `modules/<name>/` → `model.go`, `service/` or `service.go`, `repository/sqlstore/`, `transport/http.go` |
| Shared core capability | `internal/<name>/` |
| Admin UI component | `internal/ui/components/` (`.templ` files) |
| Public site markup and styles | `themes/<theme>/layouts`, `themes/<theme>/blocks`, `themes/<theme>/static` |
| Database change | new directory in `migrations/` with `migration.yaml`, `sqlite.sql`, `postgres.sql`, `mysql.sql` |
| CLI command | its own file in `cmd/hydro/<name>_cmd.go`, one `case` line in `run()` |

Follow the closest existing example. `modules/news` is the reference module
for content that needs custom behaviour; everything else should be a content
type. Content types get admin screens at `/admin/resources/<id>` and, with
`public: true`, a read API at `/api/v1/content/<id>`; YAML flags add list
search/filters, drafts and revisions (see `definition.go`).

## Rules

1. **No HTML or CSS in Go strings.** Markup goes in `.templ` files or theme
   HTML; styles go in CSS files. Existing string-built HTML (for example
   `modules/news/transport/http.go`) is legacy to migrate, not a pattern to copy.
2. **No customer-specific content in core.** Brand names, hardcoded menus,
   dates, exchange rates or demo credentials belong in a theme, seed data or
   config, not in `modules/` or `internal/`.
3. **Every migration ships all three dialects** (SQLite, PostgreSQL, MySQL)
   and never edits an already-released migration. Never delete a migration
   directory: databases that applied it refuse to start without it.
4. **No SQL dialect or cache details in services.** Keep them in
   `repository/sqlstore` and `internal/cache`.
5. **An inactive module adds nothing:** no routes, migrations, workers or assets.
6. **Keep files small.** Split a file before it passes about 800 lines.
   `cmd/hydro/main.go` is legacy; don't add to it, move code out of it.
7. **Don't write status reports, evidence logs or plans into the repo.**
   Put the explanation in the commit message. Add a decision record in
   `docs/decisions/` only when you change a module boundary or a public
   contract (routes, API, config, migrations).
8. **Never hide a failing test** by renaming it (`skip_TestX`) or adding a
   bare `t.Skip`. Fix the code, update the test to the new intended
   behaviour, or delete it with an explanation in the commit. `codeguard`
   rejects renamed tests.
9. **Regenerate, don't hand-edit**, generated files (`*_templ.go`, compiled
   CSS): `go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate`.

## UI work

- Admin UI: Go + Templ + HTMX. Build screens from the components in
  `internal/ui/components`; if one is missing, add it there rather than
  writing one-off markup.
- New admin CSS goes in `static/admin/admin.css` (served at
  `/admin-assets/admin.css`), never in Go strings. It already provides hover,
  press, focus-visible, invalid-field and htmx swap transitions on the shell's
  `--motion-*` tokens; reuse them and respect `prefers-reduced-motion`.
- htmx is loaded on every admin page. To show a toast after a request, send
  `HX-Trigger: {"hydro:toast": {"message": "Saved"}}`.
- Look at the result. Start the dev server (`go run ./tools/devserver`), then
  screenshot the pages you changed at desktop and phone width and open the PNGs:

  ```bash
  HYDRO_ADMIN_EMAIL=... HYDRO_ADMIN_PASSWORD=... node e2e/screenshot.mjs /admin/resources/faq
  ```

  It exits non-zero on HTTP errors or JavaScript errors. On Windows Git Bash,
  prefix it with `MSYS_NO_PATHCONV=1`.

## Commands

```bash
go run ./tools/devserver   # dev server with rebuilds + demo data, http://127.0.0.1:8090
go run ./tools/check       # everything CI checks that runs fast locally (or: make check)
go test ./modules/news/... # one package
```
