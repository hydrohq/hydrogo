# 0032. Fail-Closed PostgreSQL RLS, Zero-Dependency In-Memory Engine, and Storefront Resilience

Date: 2026-11-03
Status: Accepted

## Context

HydroGo aims to provide a secure, single-binary headless CMS and boutique storefront platform with zero external dependencies in its default profile. Recent audits and hardening reviews surfaced several security, performance, and operational vulnerabilities across the stack:

1. **PostgreSQL RLS Bypass Risk**: Tenant isolation policies previously relied on an `app.tenant_rls_enforced` session setting. If uninitialized or unset, queries could fail open or allow cross-tenant data leakage if application-level tenant filters were misconfigured. Furthermore, connection pooling (`pgxpool`) could retain session variables (`SET LOCAL app.current_tenant_id`) across transaction rollbacks.
2. **On-Demand TLS and Database DoS**: Caddy's `on_demand_tls` query endpoint (`/api/v1/internal/check-domain`) lacked in-memory negative caching. Rapid scans or invalid domain requests repeatedly hit the database, exposing the instance to denial-of-service and certificate exhaustion risks.
3. **External Cache Dependencies**: Rate limiting and ephemeral caching previously used either Redis or disparate local maps. Running a fully featured single-binary CMS should not require an external Redis daemon for in-memory TTL caching, atomic rate limiting, or background pub/sub workers.
4. **SSRF and AI Prompt Privacy**: AI generation endpoints (`/api/v1/admin/ai/generate` and adapters) lacked socket-level dialer controls, risking DNS rebinding attacks to loopback or RFC 1918 internal addresses. Copilot system prompts also fetched live database records, creating PII disclosure and indirect prompt injection vectors.
5. **Storefront Operational Gaps**: Storefront HTML responses lacked conditional HTTP caching (ETags / 304 Not Modified), embedded fonts were repeatedly read from disk/embed per request, and WhatsApp orders lacked persistence in the database prior to external redirect.

## Decision

1. **Fail-Closed PostgreSQL RLS & Connection Pool Discard**:
   - Replaced all tenant isolation policies in migration `202611020002_postgres_rls_fail_closed` with strict `nullif(current_setting('app.current_tenant_id', true), '')` predicates. Without an active tenant context, queries fail closed and return zero rows.
   - Configured `pgxpool.Config.AfterRelease` hook to execute `DISCARD TEMP; RESET ALL;` ensuring pooled connections cannot leak tenant session variables or temporary tables.

2. **Zero-Dependency Embedded Memory Engine (`internal/cache.MemoryStore`)**:
   - Built a pure Go, thread-safe `MemoryStore` using standard library primitives (`sync.RWMutex`, `map`, `time`).
   - Features per-key TTL, background janitor sweeps with stop channels, atomic counter increments (`Incr`), sliding rate-limiting (`RateLimit`), pub/sub message brokers (`MemoryPubSub`), and bounded worker pools (`MemoryWorkerPool`).
   - Connected `MemoryStore` to Caddy `check-domain` with a 5-minute positive/negative cache (shielding the database from invalid domain lookups).
   - Hooked application rate limiters into `MemoryRateLimiter` with isolated namespace sequences.

3. **Socket-Level SSRF Prevention & Copilot Privacy**:
   - Enforced socket-level `net.Dialer.Control` validation in AI HTTP clients to abort connections when resolved IP addresses reside in private (RFC 1918), loopback, or link-local (RFC 3927) ranges.
   - Caddyfile adds rate limiting (`interval 2m`, `burst 5`) to `on_demand_tls` and blocks external access to `/api/v1/internal/*` (`respond 403`).
   - Copilot workspace prompts strip live database records, including only schema definitions and field metadata to prevent indirect prompt injection and customer data disclosure.

4. **Storefront Performance & Order Tracking**:
   - Added conditional GET support with SHA-256 ETags and HTTP 304 responses in `internal/storefront`.
   - Preloaded embedded TTF fonts into an in-memory byte map at server start.
   - Before redirecting to WhatsApp, the storefront persists an order with `PENDING_WHATSAPP` status via `POST /api/v1/store/orders` and embeds the generated order reference code into the WhatsApp message text.
   - Added product options/variants support and a responsive slide-over cart drawer with hover crossfade for secondary product images.

5. **Automated Scheduled Backups & Active Healthchecks**:
   - Added configurable automated nightly backups (`HYDRO_BACKUP_ENABLED`, `HYDRO_BACKUP_HOUR`, `HYDRO_BACKUP_DIR`, CLI: `hydro backup auto`).
   - Updated `/healthz` to perform active database connectivity verification via `db.PingContext`.

## Consequences

- Standalone HydroGo instances gain production-grade caching, rate limiting, and pub/sub without installing or maintaining Redis.
- PostgreSQL deployments are guaranteed fail-closed against cross-tenant data leakage even under edge-case connection recycling.
- Caddy on-demand TLS endpoints are protected against database exhaustion attacks.
- Storefront traffic enjoys lower TTFB and bandwidth savings via conditional GET caching and preloaded font assets.
- Store owners retain an auditable order log of customers initiating WhatsApp checkout flows.
