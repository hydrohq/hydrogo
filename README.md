# HydroGo

**Single-binary headless CMS and application engine written in Go.**

[![Release](https://img.shields.io/github/v/release/hydrohq/hydrogo)](https://github.com/hydrohq/hydrogo/releases/latest)
[![License: PolyForm Shield 1.0.0](https://img.shields.io/badge/License-PolyForm%20Shield%201.0.0-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/hydrohq/hydrogo.svg)](https://pkg.go.dev/github.com/hydrohq/hydrogo)
[![Architecture: Single Binary](https://img.shields.io/badge/Architecture-Single%20Binary-darkgreen)](docs/decisions/0005-admin-svelte-spa.md)

HydroGo is a self-contained, high-performance headless CMS and application engine. It combines declarative YAML content modeling, real-time Server-Sent Events (SSE), an embedded Svelte 5 admin panel, and an AI-native Model Context Protocol (MCP) server into one single executable.

Run it on SQLite with zero external dependencies, or connect to PostgreSQL or MySQL for distributed workloads.

[Documentation](https://hydrogo.dev/docs) &bull; [Architecture Vision](docs/VISION.md)

---

## Architectural Highlights

```
+-------------------------------------------------------------------------+
|                           HydroGo Single Binary                         |
|                                                                         |
|  +--------------------+  +--------------------+  +-------------------+  |
|  | Embedded Admin SPA |  | REST & Realtime    |  | Native MCP Server |  |
|  | Svelte 5 + Tailwind|  | JSON API + SSE Bus |  | AI Tooling Engine |  |
|  +---------+----------+  +---------+----------+  +---------+---------+  |
|            |                       |                       |            |
|  +---------v-----------------------v-----------------------v---------+  |
|  |               Core Engine, Cache & Collection Bus                 |  |
|  |   Singleflight Dedup | ETag Caching | Rate Limiter | Pub/Sub     |  |
|  +---------------------------------+---------------------------------+  |
|                                    |                                    |
|  +---------------------------------v---------------------------------+  |
|  |                 Pluggable Storage Layer                           |  |
|  |   SQLite (WAL + 0-copy) | PostgreSQL (RLS) | MySQL / MariaDB      |  |
|  +-------------------------------------------------------------------+  |
+-------------------------------------------------------------------------+
```

- **Single Binary Simplicity:** Backend services and the modern Svelte 5 admin interface are compiled directly into a single binary via Go `embed.FS`. No Node.js runtime or NPM dependencies required in production.
- **Declarative YAML Content Modeling:** Define schemas in `content-types/*.yaml`. HydroGo automatically generates typed admin screens, REST CRUD endpoints, search filters, and real-time subscription streams.
- **Model Context Protocol (MCP) Built-In:** First-class native MCP server (`hydro mcp`) allows AI agents (Cursor, Windsurf, Claude Desktop) to inspect schemas, query content, execute workflows, and verify cryptographic audit trails.
- **Self-Healing Diagnostics (`hydro doctor`):** Automated health checks verify database schema integrity, SQLite WAL pragmas, content-type schemas, media permissions, and port availability with optional automatic repairs (`--fix`).
- **High-Performance Read/Write Pipeline:** Zero-copy database row scanning, `singleflight` query deduplication, conditional `ETag` / `304 Not Modified` caching, and optimized SQLite pragmas delivering over 2,780 reads/sec with sub-10ms latency.
- **Zero Build on Publish:** Content edits and publications take effect immediately in the database and API. No static site generator rebuilds or deployment delays.
- **Multi-Tenant Agency Isolation:** Opt-in multi-tenancy with fail-closed PostgreSQL Row-Level Security (RLS) and tenant-scoped connection pools.
- **Unslop Design Standard:** Admin UI crafted with Quiet Chrome, Zinc tokens, strict 4px layout grids, and zero visual noise.

---

## Performance Benchmarks

Measured using `tools/bench` on a standard single-core commodity VPS running SQLite in WAL mode:

| Benchmark Scenario | Concurrency | Throughput | Latency (p50) | Latency (p95) | Error Rate |
|---|---|---|---|---|---|
| **Public Content Reads** (cached + ETag) | 50 workers | **2,783.1 req/s** | **9.66 ms** | 44.42 ms | 0.00% |
| **Concurrent Record Writes** | 10 workers | **959.5 writes/s** | **7.85 ms** | 89.20 ms | 0.00% |
| **Realtime SSE Broadcast** | 20 listeners | **100% delivered** | < 1.00 ms | < 2.50 ms | 0.00% |

---

## Quick Start

### 1. Installation

Download the binary from the [latest release](https://github.com/hydrohq/hydrogo/releases/latest) or install via terminal:

**macOS / Linux:**
```bash
curl -fsSL https://github.com/hydrohq/hydrogo/releases/latest/download/install.sh | sh
```

**Windows (PowerShell):**
```powershell
irm https://github.com/hydrohq/hydrogo/releases/latest/download/install.ps1 | iex
```

**Docker:**
```bash
docker run -p 8080:8080 -v hydro-data:/app/data ghcr.io/hydrohq/hydrogo:latest
```

**From Source (Go 1.27+):**
```bash
git clone https://github.com/hydrohq/hydrogo.git
cd hydrogo
go build -o bin/hydro ./cmd/hydro
```

### 2. Scaffold a Project

Use `hydro new` to initialize a new project with your chosen domain preset:

```bash
hydro new mysite --preset saas --admin-email admin@example.com
cd mysite
```

Available presets (`hydro new --list-presets`):
- `minimal`: Bare headless core with system settings.
- `blog`: Articles, categories, tags, and authors.
- `boutique`: E-commerce catalog, product variants, and WhatsApp ordering.
- `restaurant`: Menus, categories, opening hours, reservations, and reviews.
- `saas`: Plans, pricing tiers, feature matrices, FAQs, and testimonials.
- `news`: High-throughput journalism with editorial workflows and scheduled publishing.
- `corporate`: Services, team directory, case studies, and lead intake.
- `crm`: Customer accounts, interactions, deals, and stages.

### 3. Verify System Health

Run the diagnostic suite before starting your server:

```bash
hydro doctor
# Or automatically repair safe local issues (directory permissions, SQLite pragmas):
hydro doctor --fix
```

### 4. Start the Engine

```bash
hydro serve --config config.yaml
```

- **Admin Panel:** [http://127.0.0.1:8080/admin](http://127.0.0.1:8080/admin)
- **Public REST API:** [http://127.0.0.1:8080/api/v1](http://127.0.0.1:8080/api/v1)
- **OpenAPI Documentation:** [http://127.0.0.1:8080/api/v1/openapi.json](http://127.0.0.1:8080/api/v1/openapi.json)

---

## Declarative Content Modeling

HydroGo replaces manual database migrations for application data with declarative YAML files in `content-types/`:

```yaml
# content-types/features.yaml
id: features
label: Product Features
public: true
drafts: true
revisions: true
realtime: true

fields:
  - {id: name, label: Feature Name, kind: text, required: true, list: true, searchable: true}
  - {id: slug, label: Slug, kind: text, required: true}
  - {id: summary, label: Summary, kind: text, required: true}
  - {id: tier, label: Plan Tier, kind: select, options: [free, pro, enterprise], list: true, filterable: true}
  - {id: description, label: Description, kind: richtext}
  - {id: active, label: Active, kind: boolean, list: true}
```

Saving this file automatically registers:
1. An admin CRUD panel with search, sorting, filtering, and revision history.
2. A typed REST endpoint at `GET /api/v1/content/features` with `?search=`, `?filter[tier]=`, and pagination.
3. A real-time SSE event stream at `GET /api/v1/content/features/subscribe`.

---

## Model Context Protocol (MCP)

HydroGo includes a built-in MCP server for seamless integration with AI coding assistants and autonomous agents:

```bash
# Run standalone MCP stdio server
hydro mcp --config config.yaml
```

### Cursor & Windsurf Configuration

Add HydroGo to your `.cursor/mcp.json` or Claude Desktop configuration:

```json
{
  "mcpServers": {
    "hydrogo": {
      "command": "hydro",
      "args": ["mcp", "--config", "/path/to/hydro.yaml"]
    }
  }
}
```

AI agents can automatically:
- Discover registered content types and schemas (`schema_get`).
- Query, filter, and mutate content records (`content_list`, `content_create`, `content_update`).
- Inspect runtime health and memory metrics (`system_info`).
- Verify cryptographic audit log integrity (`system_verify_audit`).

---

## Repository Structure

| Directory | Purpose |
|---|---|
| `cmd/hydro/` | CLI commands (`serve`, `new`, `doctor`, `mcp`, `migrate`, `backup`, `token`) |
| `admin/` | Svelte 5 + TypeScript + Tailwind embedded admin panel SPA |
| `internal/` | Engine core: database abstraction, collections, cache, security, MCP, media, auth |
| `modules/` | Domain modules: `pages`, `news`, `members`, `site`, `seo`, `commerce` |
| `content-types/` | Stock and user declarative YAML content definitions |
| `migrations/` | Dialect-complete SQL migrations (SQLite, PostgreSQL, MySQL) |
| `packages/client/` | `@hydrogo/client` typed TypeScript API client |
| `examples/` | Backend examples: `side-service` (daemon integration), `benchmark` (load schemas) |
| `tools/` | Development harness: `devserver`, `check` (CI runner), `bench` (load testing) |
| `docs/` | Architecture Decision Records (ADRs), deployment guides, and API documentation |

---

## Contributing

We welcome community contributions! Please review:
1. [CONTRIBUTING.md](CONTRIBUTING.md) for workflow, branch rules, and PR guidelines.
2. [AGENTS.md](AGENTS.md) for architectural invariants (800-line budget, dialect completeness, zero emojis in UI chrome).
3. Run verification before submitting code:
   ```bash
   go run ./tools/check
   ```

---

## License

[PolyForm Shield License 1.0.0](LICENSE). Free to use, modify, and run for your own applications, sites, and stores. You may not offer HydroGo as a competing managed CMS product or hosted CMS service.
