# HydroGo — Workspace/Engine for Client Editions

HydroGo is a Go-based **product factory**, not a single customer-specific CMS.
It turns versioned blueprints, modules, and policies into deployable Client
Editions for corporate sites, editorial platforms, portals, mobile backends, and
transaction-heavy business products. The Core stays generic; the business
meaning of a particular customer lives in an Edition's `custom/` boundary.

The purpose is to give teams a secure, ultra-portable, and operable foundation that
they can own and deploy without forking HydroCore or accepting infrastructure
they do not use. A Client Edition can be delivered as one self-contained Go
binary, with its database, media, configuration, and secrets cleanly separated.

HydroGo is a contract-driven runtime and workspace: inactive modules create no
routes, migrations, assets, workers, or mandatory services; customer workflows
are explicit policy code; schema changes are reviewed migrations, never runtime
DDL.

## Architecture at a glance

```text
Blueprint / Edition manifest / module contracts
                         │
                         ▼
                 generated/ artifacts
                         │      custom/ policy + workflow
                         └──────────────┐
                                        ▼
HTTP/JSON/Templ → application service → repository contract → SQLite/PostgreSQL/MySQL
                                        │
                         audit · jobs · cache · media · observability
```

The HTML admin surface and JSON API call the same application services,
validation, authorization, and transaction boundaries:
- **Compact & Shared Hosting Profile:** In-process SQLite + Native CGI/FastCGI (Runs on $10-15/yr cPanel/Plesk with 4ms TTFB).
- **Standard & Dedicated Profile:** Dedicated VPS/Container with PostgreSQL or MySQL 8 / MariaDB.
- **Pure Go Media Pipeline:** Resizing, WebP/AVIF, BlurHash/LQIP, watermarking, and focal point cropping without external FFmpeg or ImageMagick binaries.

## Key Capabilities & Features

- **Blazing Fast CGI / FastCGI Support:** Native support for shared cPanel/Plesk hosting with automated static HTML passthrough caching via `.htaccess` (0ms / 0 CPU for cached pages).
- **Type-Safe Templ Design System:** Server-side rendered UI components with Tailwind CSS tokens (Bento Grids, Accordions with automatic `FAQPage` JSON-LD, Video Players, Before/After Sliders, Hotspots).
- **Form Builder & Lead CRM:** Built-in form submission inbox, honeypot & Cloudflare Turnstile spam protection, double-submission locks, autoresponder emails, and durable SQLite mail spooler with exponential backoff retry.
- **AI-Ready & Modern SEO:** Automatic OpenGraph social image generator (Pure Go), dynamic sitemap indexing/chunking, `/llms.txt` and `/llms-full.txt` endpoints for AI web crawlers (ChatGPT, Claude, Perplexity).
- **Zero-Cookie Privacy Analytics:** GDPR/KVKK compliant server-side pageview and referrer tracking without client cookies or third-party trackers.
- **Bulletproof Security:** Progressive exponential delay (Tarpit) brute-force protection, TOTP 2FA (Google Authenticator), instant in-memory IP blocking (`hydro ban`), preset security headers (CSP/HSTS), and SQLite startup WAL recovery.
- **Zero-Jargon Admin Studio:** Audience-separated business owner view, visual translation matrix (side-by-side i18n), live diff revision history, bulk action toolbar, CSV/XLSX import wizard, and automated weekly digest emails.

## Quick Start

### Installation

Install the `hydro` CLI globally (works identically on Linux, macOS, and Windows):

```bash
go install ./cmd/hydro
```

Or build the local binary:

```bash
make build
```

### Starting a New Project

Create a new Client Edition scaffold with a pre-configured preset:

```bash
hydro new my-project --preset=corporate
cd my-project
hydro project check
```

Presets available: `corporate`, `news`, `agency`, `saas`, `minimal`.

### Running Locally

```bash
# 1. Apply database migrations
hydro migrate

# 2. Start the live development server with hot-reloading
hydro dev
```

The development server listens on `http://localhost:8090` with instant live-reload for Templ, Go code, and static assets.

### Production Deployment

#### 1. Standalone Production Server (VPS / Docker)

```bash
hydro serve --address=0.0.0.0:8080
```

#### 2. Shared Hosting ($10/yr cPanel / Plesk via CGI)

Compile the self-contained CGI binary:

```bash
make build-cgi
```

Upload `hydro.cgi` and `deploy/hosting/.htaccess` to your `public_html/` folder. Apache will automatically route dynamic requests to `hydro.cgi` and serve cached static HTML directly from disk.

## Common CLI Commands

```bash
# Development & Operations
hydro dev                             # Start local dev server with live-reload
hydro serve                           # Run standalone production HTTP server
hydro serve --cgi                     # Handle single request from CGI web server
hydro migrate                         # Apply pending database migrations
hydro doctor                          # Run system integrity and database diagnostics
hydro doctor --fix                    # Diagnose and automatically repair SQLite locks/permissions
hydro clean --demo                    # Purge disposable demo seed data

# Project & Resource Scaffolding
hydro new <project-name>              # Scaffold a new Client Edition
hydro new resource                    # Interactive Blueprint & CRUD generator
hydro sdk generate --out=sdk          # Generate TypeScript and Flutter/Dart client SDKs

# Security & Administration
hydro create-admin                    # Create a new administrator user
hydro ban <ip-or-cidr>                # Block malicious IP address
hydro unban <ip-or-cidr>              # Unblock IP address
hydro rollback                        # Roll back database to pre-migration snapshot

# Backups & Data Portability
hydro backup sqlite --destination=bak/ # Create immutable snapshot backup
hydro restore verify --snapshot=bak/   # Verify and restore backup snapshot
hydro portable export --path=data.json # Export allowlisted data document
hydro portable import --path=data.json # Import portable data document
```

## Capacity Profiles

- `compact`: A zero-dependency profile using SQLite and local media for shared hosting and minimal containers.
- `standard`: The recommended profile for dedicated servers with PostgreSQL / MySQL, Redis cache, and higher connection budgets.
- `distributed`: A multi-instance profile supporting distributed Redis L2 cache, S3-compatible media/CDN topology, and background worker pools.

## Testing & Quality Gates

Run all unit, integration, and contract tests:

```bash
go test ./...
```

Run tests with race detection:

```bash
go test -race ./...
```

Verify formatting and linting:

```bash
make verify
```

## Documentation

- [Roadmap](docs/ROADMAP.md): Phases, milestones, and acceptance criteria
- [HydroGo Philosophy](docs/HYDROGO_PHILOSOPHY.md): Product and architectural principles
- [Information Hierarchy](docs/INFORMATION_HIERARCHY.md): Client vs Developer surface guidelines
- [Architecture Decisions (ADRs)](docs/architecture/README.md): Binding architectural decisions
- [Operations Runbook](docs/OPERATIONS_RUNBOOK.md): Deployment, backup, restore, and monitoring flows
- [Component Guidelines](docs/UI_COMPONENTS.md): Design system and Templ component standards
- [Security Policy](SECURITY.md) and [Contributing Guide](CONTRIBUTING.md)

## License & Distribution

This repository is proprietary and private. Commercial distribution and licensing terms are detailed in [`PROPRIETARY_NOTICE.md`](PROPRIETARY_NOTICE.md) and [`docs/COMMERCIAL_LICENSING.md`](docs/COMMERCIAL_LICENSING.md).
