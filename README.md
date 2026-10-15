# HydroGo

**A single-binary content management system written in Go.**

HydroGo manages pages, news, media and members, renders themed public sites
with server-side HTML, and exposes the same content over a REST API. It runs
as one self-contained binary on SQLite, PostgreSQL or MySQL/MariaDB.

> Status: pre-1.0. HydroGo is being refocused from a broad "product factory"
> into a CMS. See [docs/decisions/0001-cms-focus.md](docs/decisions/0001-cms-focus.md).

## What it does

- **Content:** pages with blocks, news with categories and tags, media
  library (WebP conversion, focal-point crops, BlurHash), drafts, revisions,
  scheduled news publishing, trash and audit log.
- **Sites:** headless — build any frontend on the API (see `examples/`), navigation,
  redirects, SEO metadata, sitemaps, Open Graph images, `llms.txt`.
- **Members:** registration, login, MFA for admins, privacy-friendly
  analytics, forms with a submission inbox, weekly digest emails over SMTP.
- **Admin:** a server-rendered admin panel (Go + Templ + HTMX).
- **Headless:** a public content REST API described by OpenAPI in
  `docs/api/`, with `hydro sdk generate` for TypeScript and Dart clients.
- **Operations:** migrations with rollback snapshots, backups and restore,
  health/readiness probes, Prometheus metrics, optional Redis and S3.

## Quick start

Requirements: Go 1.27.2+ (the toolchain is pinned in `go.mod`).

```bash
# Development server with rebuild-on-save and disposable demo data
go run ./tools/devserver
# then open http://127.0.0.1:8090 and http://127.0.0.1:8090/admin
```

Production-style run:

```bash
go build -o bin/hydro ./cmd/hydro
./bin/hydro migrate --config config.yaml
printf '%s' 'a-long-password' | ./bin/hydro create-admin --email you@example.com --password-stdin --config config.yaml
./bin/hydro serve --config config.yaml
```

Copy `config.example.yaml` to start a configuration. `hydro help` lists all
commands.

## Repository layout

| Path | What lives there |
|---|---|
| `cmd/hydro` | The CLI and HTTP server entrypoint |
| `internal/` | Core packages: auth, database, media, admin UI (`ui`, `adminstudio`), collections, search, security, … |
| `modules/` | Content modules: `pages`, `news`, `members`, `site`, `seo`, `designsystem` |
| `migrations/` | SQL migrations for SQLite, PostgreSQL and MySQL |
| `extension/v1` | Theme and extension contracts |
| `tools/` | Dev server and CI guards |
| `docs/` | Current docs; `docs/archive/` holds historical plans and ADRs |

## Development

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Contributors and coding agents
must read [AGENTS.md](AGENTS.md) first.

## License

[MIT](LICENSE). Security issues: see [docs/SECURITY.md](docs/SECURITY.md).
