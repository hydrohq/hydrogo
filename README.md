# HydroGo

**A single-binary headless content management system written in Go.**

HydroGo manages pages, news, media, members, and custom content types, exposing
everything over a unified REST API with an embedded modern admin panel. It runs
as one self-contained binary on SQLite, PostgreSQL or MySQL/MariaDB.

> Status: v2 Headless. HydroGo serves pure JSON APIs and embeds a Svelte 5 SPA
> admin panel. See [docs/decisions/0001-cms-focus.md](docs/decisions/0001-cms-focus.md) and
> [docs/decisions/0005-admin-svelte-spa.md](docs/decisions/0005-admin-svelte-spa.md).

## What it does

- **Headless Core:** pure JSON REST APIs with OpenAPI schema for pages, news,
  members, media, settings, and custom YAML content types.
- **Admin Panel:** embedded single-page application built with Svelte 5,
  TypeScript, Tailwind CSS, and shadcn-svelte (`admin/` embedded in Go binary).
- **Custom Content Types:** declarative YAML definitions (`content-types/*.yaml`)
  instantly provide admin screens and `/api/v1/content/<id>` endpoints with zero code.
- **Content:** pages with blocks, news with categories and tags, media
  library (WebP conversion, focal-point crops, BlurHash), drafts, revisions,
  scheduled news publishing, and audit log.
- **Frontends:** build any client on the API (see `examples/svelte-starter`).
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
| `admin/` | Modern Svelte 5 + TypeScript + Tailwind embedded admin panel SPA |
| `internal/` | Core packages: auth, database, media, admin UI embedding (`adminui`), collections, search, security, … |
| `modules/` | Content modules: `pages`, `news`, `members`, `site`, `seo` |
| `content-types/` | Declarative YAML schemas for custom content types |
| `migrations/` | SQL migrations for SQLite, PostgreSQL and MySQL |
| `examples/` | Headless client starter implementations (`examples/svelte-starter`) |
| `tools/` | Dev server, guards, and check script |
| `docs/` | Current docs and architecture decision records (ADRs) |

## Development

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Contributors and coding agents
must read [AGENTS.md](AGENTS.md) first.

## License

[MIT](LICENSE). Security issues: see [docs/SECURITY.md](docs/SECURITY.md).
