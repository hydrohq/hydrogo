# 0029: Platform Orchestrator, Resource Governor & Multi-Tenant SaaS Control Plane

- Status: Accepted
- Date: 2026-10-31

## Context

When running HydroGo as a single-binary multi-tenant SaaS engine hosting dozens or hundreds of customer workspaces, operators need:
1. **Zero-Restart Tenant & Domain Provisioning**: Registering new workspaces, subdomains, and custom domains at runtime without editing YAML files or restarting the server process.
2. **Lifecycle & Subscription Governance**: Managing plans (`starter`, `pro`, `vip`, `custom`), trial/subscription expiration timestamps, automatic read-only degradation when a subscription expires (keeping public storefronts online while freezing admin/AI writes), and an instant Kill-Switch (`suspended` / `archived`).
3. **Noisy-Neighbor Protection (Resource Governor)**: Enforcing per-tenant storage quotas (`max_storage_bytes`), collection record limits (`max_records`), request rate limits (`rate_limit_rps`), in-flight concurrency semaphores (`max_concurrent_reqs`), upload payload limits (`max_upload_bytes`), and monthly AI token budgets (`ai_monthly_token_limit`) in-process without requiring Docker-per-tenant overhead.
4. **Caddy On-Demand TLS Verification**: Providing a fast loopback verification gate (`GET /api/v1/system/domain-check?domain=...`) so reverse proxies only issue ACME certificates for registered active domains.
5. **Audited Support Impersonation**: Allowing the platform owner to issue short-lived (15-minute), audited support tokens to inspect and assist a tenant workspace without sharing passwords.

## Decision

1. **Platform Governance Ledger (`202611010003_orchestrator_governance`)**:
   - Introduce the `tenant_governance` table across SQLite, PostgreSQL, and MySQL to store per-tenant lifecycle states, plan presets, resource quotas, custom domains, and AI metering counters.
   - Integrate `internal/tenant.GetTenantQuotas` with `tenant_governance` so storage and collection record quotas dynamically reflect the orchestrator's configuration.

2. **In-Process Resource Governor (`internal/tenant/governor.go`)**:
   - Enforce lifecycle transitions at the HTTP middleware boundary:
     - `suspended` / `archived` workspaces return `503 Service Unavailable` on tenant routes while leaving `/api/v1/admin/orchestrator/*` accessible to the platform administrator.
     - Active workspaces whose `subscription_ends_at` or `trial_ends_at` timestamp has passed automatically degrade to `readonly`, serving `GET`/`HEAD` public storefront traffic normally while rejecting mutating requests (`POST`/`PUT`/`PATCH`/`DELETE`) with `402 Payment Required`.
   - Enforce per-tenant token-bucket RPS limits and atomic in-flight request semaphores (`429 Too Many Requests`), plus per-tenant upload body caps (`http.MaxBytesReader`).

3. **Dynamic Host Resolution & Caddy TLS Gate**:
   - Upgrade `internal/tenant.HostResolver` with thread-safe runtime mutation (`SetHost`, `RemoveHost`, `SetFallback`) backed by `tenant.ResolveTenantByDomain`.
   - Expose `GET /api/v1/system/domain-check?domain=...` (restricted to loopback or `X-Orchestrator-Secret`) to validate custom domains and subdomains for Caddy On-Demand TLS.

4. **Orchestrator REST Control Plane (`/api/v1/admin/orchestrator/tenants`)**:
   - Strictly restricted to platform administrators (`isOrchestratorRequest` rejects tenant-scoped store admins).
   - Supports listing all workspaces with live usage metrics, zero-restart provisioning (`POST`), quota/plan/lifecycle updates (`PATCH`), 15-minute audited support impersonation (`POST .../impersonate`), and workspace archiving (`POST .../archive`).

## Consequences

- Hundreds of isolated customer workspaces can be hosted on a single HydroGo binary with deterministic CPU/memory/I/O isolation and sub-50ms provisioning.
- Subscription expirations gracefully freeze mutations without taking public customer storefronts offline.
- Tenant-created runtime schemas are isolated per tenant under `<content-types>/tenants/<tenant_id>/` when `RequireTenantScope` is active.
