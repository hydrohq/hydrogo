# HydroGo Product Vision & Architectural Roadmap (HydroGo 2.0)

## 1. Executive Vision: What HydroGo Is Becoming

HydroGo is evolving from a localized, editorial-only headless CMS into the **premier open-source, multi-tenant application backend and AI-agent control plane**, delivered as a **single, self-contained Go binary** with an embedded **Svelte 5 admin panel**.

We are building the missing bridge between **PocketBase's single-file developer simplicity** and **Supabase's enterprise PostgreSQL scalability**, with a first-class focus on **AI coding agent automation (Claude Code, Cursor, Codex)** and **multi-tenant digital agency infrastructure**.

```
+-----------------------------------------------------------------------------------------+
|                                    HYDROGO 2.0 CORE                                     |
+-----------------------------------------------------------------------------------------+
|                                                                                         |
|   [ 1. Multi-Tenant Enterprise Engine ]        [ 2. Deterministic AI Control Plane ]   |
|   - Single binary, multi-database               - Zero-restart runtime schema API       |
|   - Native PostgreSQL RLS isolation             - Machine-readable CLI & JSON diffs     |
|   - Host-based domain routing per tenant        - Idempotent declarative operations     |
|   - Cryptographic SHA-256 audit chains          - Typed dynamic SDK generator           |
|                                                                                         |
|   [ 3. High-Performance Relational BaaS ]      [ 4. Headless Content Core ]             |
|   - Pushdown SQL row-level access rules         - Real-time zero-build publishing       |
|   - Dialect expression indexes (SQLite/MySQL)   - Editorial drafts & revision history   |
|   - Distributed SSE real-time via Redis         - Embedded Svelte 5 + Zinc admin SPA    |
|   - Decoupled SQLite WAL read/write pools       - Media pipeline (WebP, BlurHash, S3)   |
|                                                                                         |
+-----------------------------------------------------------------------------------------+
```

---

## 2. Why We Are Pivoting: The Market Problem

Today's backend ecosystem leaves digital agencies, SaaS startups, and AI-assisted engineering teams in an architectural trap:

1. **PocketBase is brilliant but trapped in single-node SQLite**:
   PocketBase offers an extraordinary single-binary DX, but it cannot run on PostgreSQL, lacks native multi-tenancy (agencies must spin up separate containers per client), and has no enterprise editorial CMS workflows (revisions, drafts, image crops).
2. **Supabase is powerful but architecturally heavy**:
   Supabase provides PostgreSQL scale, but deploying it self-hosted requires 12+ Docker microservices, high RAM consumption, and steep operational overhead.
3. **AI Coding Agents lack a dependable backend control plane**:
   When AI agents (Claude Code, Cursor, Codex) build full-stack applications, they struggle with backends that require server reboots upon schema changes, return unstructured error messages, or lack safe, transactional rollback guarantees.
4. **Agencies face massive infrastructure sprawl**:
   A boutique digital agency managing 30 client web applications currently maintains 30 separate databases or CMS instances. They need **one unified Go binary on PostgreSQL** that isolates 30 client tenants with custom domains, separate admin views, and zero cross-tenant leakage risk.

---

## 3. The Four Strategic Pillars of HydroGo 2.0

### Pillar 1: Single-Binary Multi-Tenancy & Enterprise Compliance
* **Host-Based Multi-Tenancy**: Automatically map custom domains (`tenant-a.com`, `tenant-b.com`) to isolated database scopes using `HostResolver`.
* **PostgreSQL Row-Level Security (RLS)**: Enforce database-engine tenant isolation at the transaction level (`app.current_tenant_id`) across all static modules and dynamic collections.
* **Tamper-Evident Audit Chains**: Every administrative mutation writes to a cryptographic SHA-256 hash chain guarded by database triggers, provably verifiable via CLI and API.

### Pillar 2: Deterministic AI Control Plane (Agent-First Architecture)
* **Zero-Downtime Runtime Schema API**: AI agents can define, modify, or archive content types via `POST /api/v1/admin/content-types` without terminating or restarting the running server.
* **Precision Copilot with Strict Intent Adherence**: In-admin assistant operates strictly deterministically without unsolicited edits, respecting user design intent with unslop studio principles (Quiet Chrome, archetypal typography & colors, transactional undo).
* **Machine-Readable CLI (`--json`)**: Every CLI operation supports deterministic, structured JSON input and output (`hydro content-type diff --json`, `validate --json`, `apply --dry-run`).
* **Safe Idempotent Operations**: Declarative schema migrations with dry-run verification so autonomous agents never leave databases in inconsistent states.

### Pillar 3: High-Performance Multi-Dialect BaaS Engine
* **SQL Pushdown Access Rules**: Row-level rules (`@auth.id == record.owner_id`) are compiled into SQL `WHERE` clauses, guaranteeing accurate database-level pagination, limits, and zero memory-starvation bugs.
* **Expression Indexing Across All Dialects**: Generate expression indexes on JSON fields in SQLite (`json_extract`) and MySQL (functional indexes), matching PostgreSQL's GIN/BTree performance.
* **Distributed Real-Time Subscriptions**: SSE event hub backed by optional Redis Pub/Sub, enabling seamless real-time message broadcasting across horizontally scaled clusters.

### Pillar 4: Turnkey Headless CMS with Embedded Svelte 5 SPA
* **Embedded Modern Admin UI**: Svelte 5 runes + TypeScript + Tailwind CSS + shadcn-svelte compiled directly into the Go executable via `//go:embed`.
* **Zero Build on Publish**: Content edits are immediately live via REST APIs; frontends never require static rebuilds or redeployment.
* **Full Editorial Suite**: Block editor for pages, news with taxonomies, scheduled publishing, media processing (WebP conversion, BlurHash, focal crops), and trash recycle bin with restore controls.

---

## 4. What We Stopped Building (Disciplined Scope)

To maintain architectural focus, HydroGo has intentionally stopped the following:
* **No Domestic/Regional Couriers or Gateways in Core**: Specific shipping providers (e.g. Yurtiçi Kargo) and local payment gateways (e.g. PayTR) are decoupled into modular starter packages. Core binary maintains only generic provider interfaces.
* **No Superficial AI Buzzwords**: We do not inject superficial vector wrappers or marketing fluff into the core. Our AI focus is strictly on **making HydroGo the most reliable, deterministic, agent-operable backend engine in existence**.
* **No Feature Sprawl Without Architectural Parity**: No new domain modules will be added until dynamic collections achieve complete performance and RLS parity with static relational modules.

---

## 5. Commercial Target: The Agency & Enterprise Core

Our primary monetization path is not hosting a generic consumer BaaS cloud. Instead, we are building for:

* **Digital Agencies (Agency Edition)**: A white-label, multi-tenant distribution allowing agencies to host dozens of client sites from a single VPS or Kubernetes cluster, charging their clients retainer fees while keeping hosting costs near zero.
* **Enterprise Compliance Teams**: Regulated organizations requiring single-binary deployments with cryptographic audit trail verification, PostgreSQL RLS, and on-premise air-gapped capabilities.

---

## 6. Key Architecture Decision Records

* [ADR 0027: Realtime BaaS and Row-Level Access Rules](decisions/0027-realtime-baas-and-row-level-access-rules.md)
* [ADR 0028: AI-Agent Control Plane & Model Context Protocol (MCP) Architecture](decisions/0028-ai-agent-control-plane-and-mcp.md)
