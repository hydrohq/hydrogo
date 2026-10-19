# HydroGo Documentation

Welcome to the official HydroGo documentation. HydroGo is a high-performance, single-binary headless Content Management System written in Go, powered by an embedded modern admin panel (Svelte 5) and pure JSON REST APIs.

---

## Table of Contents

1. [Overview & Philosophy](#overview--philosophy)
2. [Quick Start](#quick-start)
3. [Architecture & Project Layout](#architecture--project-layout)
4. [Admin Studio](#admin-studio)
5. [Content Types (Declarative YAML)](#content-types-declarative-yaml)
6. [Public REST API Reference](#public-rest-api-reference)
7. [Frontend Integration](#frontend-integration)
8. [Production Deployment](#production-deployment)

---

## Overview & Philosophy

HydroGo is built around a key architectural principle:
- **Headless & Decoupled:** Go serves pure JSON APIs. No HTML/CSS markup is rendered on the server (except transactional emails).
- **Single Binary:** Everything is bundled together into one standalone binary, including database migrations, the Svelte 5 Admin Studio SPA, and the API engine.
- **Multi-Database:** Supports SQLite (zero-config local dev or small sites), PostgreSQL, and MySQL / MariaDB.
- **Extensible via YAML:** Adding new content structures (FAQs, Testimonials, Team Members, Events) requires zero Go or TypeScript code. Just create a YAML schema in `content-types/`.

---

## Quick Start

### Prerequisites
- **Go 1.27+**
- **Node.js 18+** (for frontend development)

### Running Locally with Hot Reload

Start the unified development server:

```bash
go run ./tools/devserver
```

This starts:
- The Go backend at `http://127.0.0.1:8091`
- Admin Studio Vite HMR at `http://127.0.0.1:5173/admin/`
- Svelte Starter Frontend Vite HMR at `http://127.0.0.1:3000/`
- A reverse development proxy at `http://127.0.0.1:8090/` which proxies all services under a single port!

### Local Admin Login

Navigate to [http://127.0.0.1:8090/admin/](http://127.0.0.1:8090/admin/) and log in with your demo credentials.

---

## Architecture & Project Layout

```
.
├── admin/               # Embedded Svelte 5 + TypeScript + Tailwind Admin Studio SPA
├── cmd/hydro/           # CLI entrypoints (serve, migrate, seed-demo, create-admin)
├── content-types/       # Declarative YAML schemas for custom content types
├── docs/                # Architecture decisions, guides, and API specifications
├── examples/            # Example client frontends (examples/sveltekit-starter)
├── internal/            # Core CMS engine: auth, database, security, media, api
├── migrations/          # Tri-dialect SQL migrations (SQLite, PostgreSQL, MySQL)
├── modules/             # Built-in CMS modules (pages, news, members, site)
└── tools/               # Local devserver, code guards, and test runners
```

---

## Admin Studio

Admin Studio is an embedded Single Page Application located under `/admin`.
- **Stack:** Svelte 5 (Runes), TypeScript, Tailwind CSS, shadcn-svelte.
- **Features:**
  - Content editing with rich blocks and markdown
  - Media library with automatic WebP conversion, thumbnail generation, and focal point cropping
  - Member management and authentication logs
  - Granular Role-Based Access Control (RBAC)
  - Dark / Light theme support

---

## Content Types (Declarative YAML)

To create a new content type, simply create a YAML file in `content-types/<id>.yaml`.

### Example: `content-types/faq.yaml`

```yaml
id: faq
name: FAQ
singular: Question
plural: Questions
public: true
list:
  search: [question, answer]
  default_sort: -created_at
fields:
  - id: question
    name: Question
    type: text
    required: true
    localized: true
  - id: answer
    name: Answer
    type: rich_text
    required: true
    localized: true
  - id: category
    name: Category
    type: select
    options: [General, Billing, Technical]
```

HydroGo automatically creates:
1. Admin Studio CRUD screens under `/admin/content/faq`
2. Public REST API endpoint: `GET /api/v1/content/faq`

---

## Public REST API Reference

All successful API responses adhere to the standard envelope:

```json
{
  "data": [...],
  "meta": {
    "total": 42,
    "page": 1,
    "limit": 20
  }
}
```

### Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/pages` | List published pages (`?locale=en`) |
| `GET` | `/api/v1/pages/by-slug` | Fetch page by slug (`?locale=en&slug=about`) |
| `GET` | `/api/v1/news` | List published news articles (`?locale=en&limit=10`) |
| `GET` | `/api/v1/news/by-slug` | Fetch news article by slug (`?locale=en&slug=...`) |
| `GET` | `/api/v1/content/{type}` | Fetch custom content items (e.g. `/api/v1/content/faq`) |
| `GET` | `/media/{storage_key}` | Public media assets (images, PDFs) |
| `GET` | `/readyz` | Health & readiness probe (`200 OK ready`) |

---

## Frontend Integration

Use [`@hydrogo/client`](../packages/client) (typed, with timeouts and error ids) or plain `fetch`. Public endpoints return `{data, metadata}`:

```typescript
import { createHydroGoClient } from '@hydrogo/client';

const hydro = createHydroGoClient({ origin: 'https://cms.example.com' });
const news = await hydro.listNews('en', 10);          // NewsItem[]
const faq = await hydro.listContent('faq', { limit: 20 }); // any content type
```

A complete server-rendered site is in [`examples/sveltekit-starter`](../examples/sveltekit-starter).

---

## Production Deployment

### Building the Binary

```bash
go build -o bin/hydro ./cmd/hydro
```

### Running in Production

```bash
./bin/hydro migrate --config config.yaml
./bin/hydro serve --config config.yaml
```

HydroGo supports both standard HTTP reverse-proxies (Nginx, Caddy, Cloudflare) and FastCGI (`--fastcgi`) environments.

More: [getting-started.md](getting-started.md), [deployment.md](deployment.md), [upgrading.md](upgrading.md).
