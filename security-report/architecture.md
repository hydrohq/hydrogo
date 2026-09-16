# HydroGo security reconnaissance — 2026-09-16

## Scope and method

Whole-repository, source-only review with bounded local checks. No live
endpoint, credential, external provider, or shared infrastructure was probed.
The repository is a Go modular monolith with embedded HTML assets, JSON/GraphQL
transports, a CLI, SQLite/PostgreSQL adapters, optional Redis/Valkey, media,
webhooks, and generated frontend/API artifacts.

## Entry points

- `cmd/hydro`: migrate, serve, doctor, backup/restore, project/resource
  generation and validation, license and upgrade commands.
- HTTP: public pages/news/site, member authentication, admin studio, media,
  commerce, GraphQL, health/readiness, and versioned API routes.
- Background boundaries: SQL-backed jobs/outbox, webhook delivery, media
  derivative processing, and optional distributed cache/rate limiting.
- Build/deployment: GitHub Actions, Docker/Compose definitions, release
  manifest/signature tooling, and WSL/local verification scripts.

## Trust boundaries

1. Anonymous/public browser → public read routes and bounded form endpoints.
2. Member/browser session → account, cart, checkout and personal data.
3. Admin/editor session → mutations, publishing, media, commerce and audit.
4. External provider → SMTP, payment, object storage/CDN, Redis and webhooks.
5. Operator/deployment → config, secrets, migrations, backup/restore and
   signed artifacts.
6. Generated client/customer code → versioned Core and module contracts.

## Positive controls observed

Default-deny route/policy wiring, CSRF tokens, secure-cookie/HSTS config
guards, bounded request bodies and query inputs, HTML escaping/sanitization,
redirect and webhook URL validation, path-safe media access, parameterized SQL,
bounded GraphQL query budgets, request IDs, security headers, signed release
and artifact checksums, migration compatibility checks, and non-root container
definition are present and covered by tests or static guards.
