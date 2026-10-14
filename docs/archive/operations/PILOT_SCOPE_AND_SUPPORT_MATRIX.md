# HydroGo Pilot Product Scope & Support Matrix

> **Document status:** Operational Decision Contract (Phase 0)  
> **Target Release:** HydroGo Single-Customer Production Pilot  
> **Deployment Architecture:** Isolated Per-Customer Deployment (`standard` profile)

---

## 1. Product Scope & Customer Profiles

HydroGo's initial production deployment targets a **single tenant/customer per installation** to ensure complete data sovereignty, zero multi-tenant leakage risk, and minimal operational overhead.

### Supported Customer Editions & Archetypes

| Archetype | Enabled Core Modules | Database & Storage | Workload Characteristics |
| :--- | :--- | :--- | :--- |
| **Corporate Portal & Showcase** | `pages`, `site`, `media`, `forms` | PostgreSQL 18 + Object Storage | Read-heavy (95/5), static cache hits, lead forms |
| **News & Publishing Desk** | `pages`, `news`, `site`, `media` | PostgreSQL 18 + Valkey/Redis + S3/CDN | Scheduled publishing, high concurrent reads, editorial workflow |
| **Catalog & Digital Showcase** | `pages`, `site`, `media`, `taxonomy`, `search` | PostgreSQL 18 + Valkey/Redis + S3/CDN | Faceted search, rich media gallery, high peak traffic |
| **Commerce & Member Services** | `pages`, `members`, `commerce`, `media`, `forms` | PostgreSQL 18 + Valkey/Redis + S3/CDN | Transactional consistency, payment webhooks, customer orders |

### Inactive Module Isolation Contract
Per `docs/HYDROGO_PHILOSOPHY.md` and `ADR-0033`:
- Inactive modules register **0 routes**, run **0 migrations**, create **0 worker queues**, and consume **0 background resources**.
- Each installation selects its active modules explicitly via configuration (`HYDRO_MODULES_ENABLED`).

---

## 2. Infrastructure & Capacity Profile

### Initial Live Profile: `standard`
- **Primary Database:** Managed PostgreSQL 18 (with connection pooling via PgBouncer or native tuned pool).
- **Distributed Cache & Rate Limiting:** Redis 7+ or Valkey 8 (L2 cache, atomic token bucket limiters, shared session revocation).
- **Media Asset Storage:** S3-compatible object storage (AWS S3, Cloudflare R2, MinIO) with CDN edge caching for public derivatives.
- **Worker & Outbox:** Durable transactional outbox and background job worker pool within the isolated instance boundary.

### Customer Isolation Model
- **No Shared Database:** Every client receives an independent database (`hydro_<customer_id>`), dedicated credentials, and an isolated S3 bucket / prefix (`hydro-media-<customer_id>`).
- **No Shared Process:** HydroGo runtime binary runs in an isolated container/VM instance with resource limits (CPU/Memory).
- **No Tenant Leakage Risk:** Authentication, authorization, file storage, and audit logs are physically isolated.

---

## 3. Service Level Agreements (SLA) & Operating Windows

### Availability SLA
- **Target Availability:** **99.9%** monthly uptime (maximum allowable unplanned downtime: ~43.8 minutes/month).
- **Measurement Point:** Public reverse proxy edge TLS endpoint health check (`/healthz` and `/readyz`).

### Support Hours & Incident Classification

| Severity Level | Definition | First Response Time | Resolution / Workaround Target | Support Coverage |
| :--- | :--- | :--- | :--- | :--- |
| **P1 - Critical Outage** | Production system down, data corruption risk, or total revenue/traffic loss | < 15 minutes | < 2 hours | 24/7/365 |
| **P2 - Major Degradation** | Core feature impaired (e.g., checkout or article publishing failing), workaround exists | < 1 hour | < 8 hours | 08:00 - 20:00 UTC+3 |
| **P3 - Minor Issue** | Administrative UX glitch, minor non-blocking bug | < 4 hours | Next maintenance release | Business days (09:00 - 18:00) |
| **P4 - Feature Request** | Non-urgent request or enhancement proposal | < 1 business day | Roadmap evaluation | Business days (09:00 - 18:00) |

### Maintenance Windows
- **Standard Maintenance Window:** Sunday 02:00 – 04:00 UTC+3.
- **Advance Notification:** At least 72 hours prior to scheduled non-emergency maintenance.
- **Zero-Downtime Deployments:** Routine application releases utilize blue/green binary rotation with reverse-proxy slot switching, incurring 0 HTTP interruption.

---

## 4. Data Retention, Backup & Erasure Schedule

| Data Category | Storage Location | Retention Period | Purge / Erasure Mechanism |
| :--- | :--- | :--- | :--- |
| **Published Content & Media** | PostgreSQL & S3 | Indefinite (or client-defined) | Immutable revisions until explicit admin purge |
| **Form Inbox & Inquiries** | PostgreSQL | 90 days default (configurable) | Automated scheduled job with hash-chained erasure log |
| **Audit Logs** | PostgreSQL (append-only) | 365 days | Archived to cold storage before deletion |
| **Database Snapshots** | Object Storage / Escrow | 30 days daily, 12 months monthly | Cryptographically signed, encrypted with AES-256-GCM |
| **Session & Cache Keys** | Valkey / Redis | 7–30 days max | TTL expiry + instant revocation on password/MFA reset |

---

## 5. Web Frontend Topology

- **Architecture:** The web frontend runs as an independent application consuming HydroGo's versioned `/api/v1` REST/JSON API.
- **Domain & Routing:**
  - Option A (Same-Site / Reverse-Proxy - Recommended): `example.com` (Frontend) and `example.com/api/v1` (HydroGo Backend) routed via reverse proxy (Caddy/Nginx).
  - Option B (BFF / Separate Domain): `www.example.com` (Frontend with BFF) and `api.example.com` (HydroGo API with strict non-credentialed CORS allowlist).
