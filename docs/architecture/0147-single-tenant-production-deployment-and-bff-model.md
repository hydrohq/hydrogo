# ADR-0147: Single-Tenant Production Deployment and Web BFF Model

- **Status:** Accepted
- **Deciders:** HydroGo Engineering & Architecture
- **Date:** 2026-09-19

## Context

When moving HydroGo into commercial production, deciding between shared-database multi-tenancy vs. isolated per-customer single-tenant deployments impacts security, compliance (KVKK/GDPR), operational complexity, and blast radius.

Furthermore, how the web frontend interacts with HydroGo (same-site reverse proxy, dedicated Backend-For-Frontend, or cross-origin API) determines the security posture around cookies, CORS, and token handling.

## Decision

1. **Phase 1 Production Topology:** Every commercial client receives a dedicated HydroGo instance, a dedicated PostgreSQL database (`hydro_<customer>`), and an isolated S3 bucket / prefix (`hydro-media-<customer>`).
2. **Standard Live Profile:** Production runs the `standard` profile using PostgreSQL + Redis/Valkey + S3-compatible Object Storage.
3. **Web Frontend Integration Model:**
   - **Preferred Path (Same-Site Reverse Proxy):** The public reverse proxy (e.g., Caddy/Nginx) serves both the Web Frontend and HydroGo under the same apex or subdomain (e.g., `example.com` for web, `example.com/api/v1` for HydroGo API, `example.com/admin` for HydroGo Studio).
   - **Alternative Path (BFF):** If the frontend resides on a separate domain (e.g., `www.example.com` and `api.example.com`), a Backend-For-Frontend (BFF) must handle session cookies server-side. HydroGo rejects credentialed cross-origin CORS (`Access-Control-Allow-Credentials: true` with wildcard or broad origins is strictly forbidden).
4. **Inactive Module Isolation:** If a client does not require e-commerce or member portals, those modules remain disabled with 0 routes, 0 migrations, and 0 workers.
5. **Multi-Tenant SaaS Boundary:** Shared-database multi-tenancy is explicitly deferred to Phase 6 as a separate architectural milestone per ADR-0140.

## Consequences

- **Positive:** Zero risk of cross-tenant data leaks; compliance and data deletion requests are physically contained; predictable blast radius and noisy-neighbor isolation; straightforward blue/green deployment and rollback per client.
- **Negative:** Infrastructure orchestration overhead per tenant (mitigated via standardized IaC/Docker/Kubernetes manifests).
