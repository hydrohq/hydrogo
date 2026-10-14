# Local development setup (AI guide)

Audience: an agent working inside a HydroGo **Core checkout** (a repo with
`go.mod` and `tools/devserver` — this repo, `C:\dev\cms`, is one). This is
not the workflow for a generated Client Edition project (`hydro project new`
output) — that's a separate, simpler `hydro run` path documented in
`docs/architecture/0337-hydro-dev-client-edition-entrypoint.md`; check
`detectDevMode` in `cmd/hydro/main.go` if unsure which kind of checkout
you're in.

## Start the dev server

```sh
go run ./cmd/hydro run
```
(`hydro dev` is an accepted alias for `run`.) Under the hood this runs
`go run ./tools/devserver`, which:

- Starts a **dev proxy** on `0.0.0.0:8090` in front of an internal app
  process on `127.0.0.1:8091`. Browse `http://127.0.0.1:8090`, not 8091.
- Defaults the database to SQLite at `data/demo.sqlite` (pass
  `--database-dsn` to override, e.g. to point at a local Postgres/MySQL).
- Runs `migrate` against that DSN automatically before first serving (with
  a retry-after-stop if SQLite reports locked, since the previous child
  process may still hold it).
- If any `.templ` source files exist, auto-installs the pinned
  `github.com/a-h/templ/cmd/templ@v0.3.1020` CLI, runs `templ generate`
  once, then keeps `templ generate -watch` running for the whole session —
  **you do not need to run `templ generate` yourself** while `hydro run` is
  active.
- Polls the filesystem (600ms default, `--poll` to change) and rebuilds +
  restarts the child binary on any Go source change. While rebuilding, the
  proxy serves a holding page ("DEV COMPILER / Updating Server") instead of
  erroring the browser.
- Seeds disposable demo content by default (`seed-demo --confirm-demo`)
  unless you pass `--seed-demo=false`. This creates a demo **member**
  account, deliberately **not** an admin account — see the companion
  `admin-panel-access` guide to get into Admin Studio.

Useful flags: `--address` (proxy bind), `--app-address` (internal app
bind), `--redis-url`, `--dependencies auto|none|required` (local Docker
Compose storage deps), `--compose-file`.

## Making a change and verifying it

1. Edit the relevant Go/`.templ` file.
2. If `hydro run` is active, it rebuilds automatically — watch its stdout
   for compile errors; a failed rebuild keeps serving the last-good build
   and reports the error in the holding page.
3. For a one-off check without the watcher: `go build ./...` and
   `go vet ./...` are fast; run the specific package's tests
   (`go test ./modules/<x>/...`) rather than the full suite while iterating.
4. Before considering a change done, check `AGENTS.md` at the repo root for
   this project's own contribution rules (ADR requirements, moduleguard/
   migrationguard/importguard gates) — they are mandatory CI gates, not
   suggestions.

## Project layout orientation

- `cmd/hydro/` — the CLI entrypoint and every subcommand (`serve`, `run`,
  `migrate`, `create-admin`, `seed-demo`, etc.) live in one large
  `main.go` plus topic-split files.
- `internal/` — engine-owned code not meant to be imported by a generated
  Client Edition (security, database, server wiring, UI shell).
- `modules/` — optional capabilities (`pages`, `news`, `members`, `media`,
  `site`, etc.), each typically with its own `transport/`, `repository/`,
  and service layer.
- `themes/` — public-facing theme HTML/CSS/JS (see the
  `frontend-development` guide).
- `migrations/` — one directory per migration, with dialect-specific SQL
  files (`sqlite.sql`, `postgres.sql`, `mysql.sql`) plus a `migration.yaml`.
- `docs/architecture/` — ADRs; read the ones relevant to what you're
  touching before changing behavior (`AGENTS.md` lists this as a hard
  requirement for architecture-affecting changes).

## Common pitfalls

- Editing a `_templ.go` file directly instead of its `.templ` source — it's
  generated, your edit is overwritten on the next `templ generate` (manual
  or watcher-triggered). Always edit the `.templ` file.
- Running the full `go test ./...` on every tiny change — slow; prefer the
  specific package while iterating, full suite before finishing.
- Forgetting `hydro run` already migrates the dev DB — running `hydro
  migrate` again manually against the same DSN is harmless but redundant.
