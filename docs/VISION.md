# HydroGo Architecture & Roadmap (HydroGo 2.0)

## 1. Overview: Where HydroGo Is Going

HydroGo is a **single-binary headless CMS, multi-tenant application backend, and AI-agent control plane** written in Go, with an embedded **Svelte 5 admin panel**.

It combines **single-binary developer simplicity** (SQLite out of the box, zero external runtime dependencies) with **PostgreSQL scalability and row-level isolation**, plus first-class tooling for **AI coding agents (Claude Code, Cursor, Codex)** via MCP and deterministic CLI commands.

```
+-----------------------------------------------------------------------------------------+
|                                    HYDROGO 2.0 CORE                                     |
+-----------------------------------------------------------------------------------------+
|                                                                                         |
|   [ 1. Multi-Tenant Engine ]                   [ 2. Deterministic AI Control Plane ]   |
|   - Single binary, multi-database               - Runtime schema API & MCP server       |
|   - Native PostgreSQL RLS isolation             - Machine-readable CLI & JSON diffs     |
|   - Host-based domain routing per tenant        - Idempotent declarative operations     |
|   - Cryptographic SHA-256 audit chains          - Typed dynamic SDK generator           |
|                                                                                         |
|   [ 3. High-Performance Relational BaaS ]      [ 4. Headless Content Core ]             |
|   - Pushdown SQL row-level access rules         - Real-time zero-build publishing       |
|   - Dialect expression indexes (SQLite/MySQL)   - Editorial drafts & revision history   |
|   - Distributed SSE real-time via Redis         - Embedded Svelte 5 admin SPA           |
|   - Decoupled SQLite WAL read/write pools       - Media pipeline (WebP, BlurHash, S3)   |
|                                                                                         |
+-----------------------------------------------------------------------------------------+
```

---

## 2. The Architectural Problem

Backend teams and agencies often have to choose between two extremes:

1. **Single-node embedded backends:** Great single-binary developer experience, but tied to SQLite without PostgreSQL scaling, native multi-tenancy, or editorial CMS workflows (revisions, drafts, image crops).
2. **Heavy multi-container stacks:** Full PostgreSQL power, but self-hosting requires a dozen microservices, higher RAM overhead, and complex operational wiring.
3. **Agent-hostile APIs:** AI coding agents struggle with backends that require manual restarts after every schema change, return unstructured errors, or lack dry-run/rollback safety.
4. **Multi-site sprawl:** Managing dozens of client sites as separate containers and databases wastes resources when one PostgreSQL-backed binary with strict tenant isolation can serve them safely.

---

## 3. The Four Strategic Pillars of HydroGo 2.0

### Pillar 1: Single-Binary Multi-Tenancy & Compliance
* **Host-Based Multi-Tenancy**: Automatically map custom domains (`tenant-a.com`, `tenant-b.com`) to isolated database scopes using `HostResolver`.
* **PostgreSQL Row-Level Security (RLS)**: Enforce database-engine tenant isolation at the transaction level (`app.current_tenant_id`) across all static modules and dynamic collections.
* **Hash-Chained Audit Logs**: Administrative mutations write to a cryptographic SHA-256 hash chain guarded by database triggers, verifiable via `hydro audit verify` and the API.

### Pillar 2: Deterministic AI Control Plane (Agent-First Architecture)
* **Runtime Schema & MCP Control Plane**: AI agents can inspect, define, and update content types via MCP (`schema_apply`) or `hydro content-type apply` without manual server restarts.
* **Precision Copilot**: Optional in-admin assistant tools operate deterministically with explicit user confirmation, consistent typography/color tokens, and transactional undo.
* **Machine-Readable CLI (`--json`)**: Key CLI operations support structured JSON output (`hydro content-type diff --json`, `validate --json`, `apply --dry-run`).
* **Safe Idempotent Operations**: Declarative schema migrations with dry-run verification so autonomous agents leave databases in consistent states.

### Pillar 3: High-Performance Multi-Dialect BaaS Engine
* **SQL Pushdown Access Rules**: Row-level rules (`@auth.id == record.owner_id`) compile into SQL `WHERE` clauses, guaranteeing accurate database-level pagination and limits.
* **Expression Indexing Across All Dialects**: Generate expression indexes on JSON fields in SQLite (`json_extract`) and MySQL (functional indexes), matching PostgreSQL's GIN/BTree performance.
* **Distributed Real-Time Subscriptions**: SSE event hub backed by optional Redis Pub/Sub for real-time updates across horizontally scaled instances.

### Pillar 4: Turnkey Headless CMS with Embedded Svelte 5 SPA
* **Embedded Modern Admin UI**: Svelte 5 runes + TypeScript + Tailwind CSS + shadcn-svelte compiled directly into the Go executable via `//go:embed`.
* **Zero Build on Publish**: Content edits are immediately live via REST APIs; frontends never require static rebuilds or redeployment.
* **Full Editorial Suite**: Block editor for pages, news with taxonomies, scheduled publishing, media processing (WebP conversion, BlurHash, focal crops), and trash recycle bin with restore controls.

---

## 4. Disciplined Core Scope

To keep the open-core binary small, portable, and maintainable:
* **No Domestic/Regional Couriers or Gateways in Core**: Specific shipping providers and regional payment gateways live in separate add-on modules. Core maintains only clean provider interfaces.
* **No Superficial AI Wrappers**: Core focuses on deterministic, verifiable, agent-operable control planes rather than generic vector/chat wrappers.
* **Architectural Parity**: Dynamic collections share the same performance, indexing, and PostgreSQL RLS guarantees as static relational modules.

---

## 5. Key Architecture Decision Records

* [ADR 0027: Realtime BaaS and Row-Level Access Rules](decisions/0027-realtime-baas-and-row-level-access-rules.md)
* [ADR 0028: AI-Agent Control Plane & Model Context Protocol (MCP) Architecture](decisions/0028-ai-agent-control-plane-and-mcp.md)
