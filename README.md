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
- **Zero build on publish:** a publish in the admin is live on the next API
  request; frontends never need a rebuild.
- **Project scaffolding:** `hydro new mysite --preset blog` creates a ready
  project (config, content types, database, admin user) in one command.
- **Frontends:** a typed client ([`@hydrogo/client`](packages/client)) and a
  server-rendered SvelteKit starter ([`examples/sveltekit-starter`](examples/sveltekit-starter)).
- **Operations:** migrations with rollback snapshots, backups and restore,
  health/readiness probes, Prometheus metrics, optional Redis and S3.

## Install

Download from the [latest release](https://github.com/ozyildirims/hydrogo/releases/latest):

| Platform | How |
|---|---|
| Windows | Run `HydroGo-Setup-<version>-windows-amd64.exe` (or `arm64`). It installs `hydro` and adds it to PATH; open a new terminal afterwards. |
| Windows (PowerShell) | `irm https://github.com/ozyildirims/hydrogo/releases/latest/download/install.ps1 \| iex` |
| Linux / macOS | `curl -fsSL https://github.com/ozyildirims/hydrogo/releases/latest/download/install.sh \| sh` |
| Docker | `docker run -p 8080:8080 -v hydro-data:/app/data ghcr.io/ozyildirims/hydrogo:latest` (see [deployment](docs/deployment.md)) |
| From source | `go build -o bin/hydro ./cmd/hydro` (Go 1.27.2+) |

Checksums are in `SHA256SUMS` on every release; the install scripts verify them.

## Quick start

```bash
hydro new mysite --preset blog --admin-email you@example.com
cd mysite
hydro serve --config config.yaml
# admin: http://127.0.0.1:8080/admin   API: http://127.0.0.1:8080/api/v1
```

`hydro new` writes `config.yaml`, the preset's `content-types/`, creates the
SQLite database and the admin user (a generated password is printed once, or
pass `--admin-password-stdin`). Presets: `minimal`, `blog`, `corporate`, `news`,
`docs` (`hydro new --list-presets`). `hydro init` does the same in the current
directory. Without `--admin-email` it only writes files and prints the next
commands.

Add a content type: `hydro content-type new recipes`, edit the YAML,
`hydro content-type validate`, restart. Then build your site on the API:
start from [`examples/sveltekit-starter`](examples/sveltekit-starter) or use
[`@hydrogo/client`](packages/client).

More: [getting started](docs/getting-started.md), [content types](docs/content-types.md),
[deployment](docs/deployment.md), [upgrading and versions](docs/upgrading.md),
[all docs](docs/README.md).

### Working on HydroGo itself

```bash
go run ./tools/devserver   # rebuild-on-save server with demo data
# then open http://127.0.0.1:8090 and http://127.0.0.1:8090/admin
```

Copy `config.example.yaml` for the full list of options. `hydro help` lists all
commands.

## Repository layout

| Path | What lives there |
|---|---|
| `cmd/hydro` | The CLI and HTTP server entrypoint |
| `admin/` | Modern Svelte 5 + TypeScript + Tailwind embedded admin panel SPA |
| `internal/` | Core packages: auth, database, media, admin UI embedding (`adminui`), collections, search, security, scaffolding (`scaffold`), … |
| `modules/` | Content modules: `pages`, `news`, `members`, `site`, `seo` |
| `content-types/` | Declarative YAML schemas for custom content types |
| `migrations/` | SQL migrations for SQLite, PostgreSQL and MySQL |
| `packages/client` | `@hydrogo/client`, the typed API client for frontends |
| `examples/` | Frontend starters: `sveltekit-starter` (SSR), plus HydroGo's own landing/docs site and a news-portal reference |
| `deploy/`, `scripts/` | Docker/Caddy/nginx/hosting examples, install scripts, Windows installer script |
| `tools/` | Dev server, guards, and check script |
| `docs/` | Current docs and architecture decision records (ADRs) |

## Development

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) and [CONTRIBUTING.md](CONTRIBUTING.md).
Coding agents must read [AGENTS.md](AGENTS.md) first.

## License

[MIT](LICENSE). Security issues: see [docs/SECURITY.md](docs/SECURITY.md).
