# 0006: Freeze legacy peripherals and native view pipeline

- Status: Accepted
- Date: 2026-10-19

## Context

Following the core refocus in ADR 0001, HydroGo bounded its scope as a single-binary
headless Go CMS. However:
1. Legacy commerce migrations (`migrations/*_commerce_*`) and unwired contracts
   (`internal/contracts/customerorders`) remained present in the codebase.
2. In the news frontend starter (`examples/hydropost-news-starter`), article view
   tracking relied on Node.js SvelteKit `+server.ts` routes reading and writing a local
   `data/views.json` file. In static SPA deployments (`adapter-static`), these server
   routes cannot execute on static web hosts (e.g. Apache/FastCGI), leading to missing
   endpoints and concurrency hazards.

## Decision

1. **Permanently Freeze Legacy Peripherals**:
   - The commerce tables and migration files remain frozen in place to satisfy database
     compatibility (Rule 3: applied databases refuse to boot if migration files are removed).
     Unselected modules are excluded by `ApplyMigrationsForModules`.
   - `internal/contracts/customerorders` is marked as frozen/deprecated.
   - Niche peripherals (`push`, `geoip`, `ai`) remain strictly optional and disabled by default.

2. **Native Aggregate-Safe View Pipeline in Go**:
   - Implemented public view tracking endpoints in Go backend:
     `POST /api/v1/news/views` (and alias `/api/views`)
     `GET /api/v1/news/views` (and alias `/api/views`)
   - Backed by the existing `analytics_content_daily` daily aggregate schema with atomic
     upserts (`INSERT ... ON CONFLICT DO UPDATE SET view_count = view_count + 1`).
   - Completely aggregate-safe and privacy-preserving: no IP, user identity, or tracking
     cookies are stored.
   - Both the public frontend and the embedded Svelte admin panel share the exact same
     view counts.
   - Removed `data/views.json` and legacy `+server.ts` view handlers from the frontend starter.

3. **Product Line Baseline (v0.1.0)**:
   - Mark the boundary with formal `CHANGELOG.md` and release tag `v0.1.0`.

## Consequences

- Frontend clients track article views via `/api/v1/news/views` directly through the Go API.
- Zero server-side file lock contention on views.
- Clean separation between core headless CMS capabilities and external consumer frontends.
