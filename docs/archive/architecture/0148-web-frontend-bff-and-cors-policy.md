# ADR-0148: Web Frontend Decoupling, BFF Architecture and CORS Policy

- **Status:** Accepted
- **Deciders:** HydroGo Engineering & Architecture
- **Date:** 2026-09-19

## Context

Decoupling the Web Frontend (Next.js, Astro, Remix, or SvelteKit) from HydroCore's monolithic server requires explicit boundaries for:
1. Versioned HTTP API consumption (`/api/v1/*`).
2. Authentication sessions (browser cookies vs tokens).
3. Cross-Origin Resource Sharing (CORS) security posture.
4. Server-Side Rendering (SSR) SEO, dynamic sitemaps, canonical meta tags, and live preview tokens.

## Decision

1. **Strict Versioned API Consumption:**
   - Headless frontends must exclusively consume versioned endpoints under `/api/v1/pages`, `/api/v1/news`, `/api/v1/forms`, etc.
   - Unversioned internal endpoints or direct SQL queries are strictly prohibited.
2. **CORS Security Posture:**
   - **Public Read Endpoints:** Open to configured frontend origin allowlists (`HYDRO_CORS_ALLOWED_ORIGINS`).
   - **No Credentialed Cross-Origin CORS:** `Access-Control-Allow-Credentials: true` with wildcard (`*`) or broad origin matching is forbidden.
3. **Session & Authentication Boundaries:**
   - When the web frontend is hosted on a different apex domain, a **Backend-For-Frontend (BFF)** pattern must be deployed.
   - The BFF holds secure session cookies with the browser, communicates with HydroGo via backend-to-backend channels, and strips browser credentials before calling HydroGo API.
   - On same-site reverse proxy setups (`example.com` and `example.com/api/v1`), standard `SameSite=Lax` secure HTTP-only cookies are used.
4. **Draft Previews & SEO:**
   - Previewing unreleased drafts requires short-lived HMAC-signed preview tokens (`?preview_token=...`) passed to the frontend renderer.
   - Sitemap generation, JSON-LD structured schema, canonical URLs, and OpenGraph meta tags are rendered directly by the frontend using metadata provided by the `/api/v1` contract.
5. **Continuous Contract Verification:**
   - Release pipelines must execute `tools/openapidiff` against the published baseline to guarantee no breaking API regressions reach production.

## Consequences

- Clean separation between frontend presentation and backend business rules.
- Eliminates Cross-Site Request Forgery (CSRF) vulnerabilities arising from permissive cross-origin credential sharing.
- Guarantees backward compatibility for headless consumers across releases.
