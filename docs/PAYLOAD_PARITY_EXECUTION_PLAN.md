# Developer Velocity & Payload Parity Execution Plan

**Status:** Partially implemented; refer to [`IMPLEMENTATION_STATUS.md`](IMPLEMENTATION_STATUS.md) for current verified status.  
**Objective:** Deliver the rapid developer workflow of modern schema-driven frameworks ("single definition -> functioning admin, API, and types") while preserving HydroGo's architectural isolation, compile-time type safety, and zero-runtime-dependency Go engine.

---

## Executive Summary

HydroGo provides a resilient core: `hydro new`, `hydro module create`, `hydro resource create`, certified SQLite/PostgreSQL contracts, draft/revision/publish editorial workflows, policy boundaries, OpenAPI v3/TypeScript generation, and multi-database adapters.

The developer velocity initiative bridges reviewable scaffolding into instant, schema-driven Admin Studio surfaces and versioned API endpoints, eliminating manual wiring without surrendering human control over authorization policies and database migrations.

---

## Phase 0 — Measurable Targets & Reference Fixture

- Define a standard multi-language editorial resource.
- Benchmark identical "first runnable admin + API + seed + test" journeys across comparison frameworks.
- Record wall-clock development duration, generated artifact count, manual file touches, memory footprint, binary size, and TTFB latency.
- Validate benchmarks across independent client editions.

**Exit Criteria:** Standard 30-minute onboarding journey and reproducible benchmark report ([`PAYLOAD_BENCHMARK_PROTOCOL.md`](PAYLOAD_BENCHMARK_PROTOCOL.md)).

---

## Phase 1 — Single-Source Resource Contract & CLI

- Consolidate fields, relations, validations, indexes, access policies, admin manifests, workflows, localization, media, and API definitions into declarative Blueprint manifests.
- Unify commands into a cohesive CLI workflow: `hydro project new`, `hydro resource new`, `hydro generate`, `hydro check`.
- Deterministically generate migrations, models, repositories, transports, policies, OpenAPI contracts, TypeScript clients, and test fixtures from Blueprint declarations.
- Wire generated routes, APIs, and Admin registrations fail-closed; never guess business policies.
- Provide `hydro resource check`, `hydro schema diff`, `hydro api diff`, and `hydro permissions check`.

---

## Phase 2 — Schema-Driven Admin Studio Engine

- Dynamically register reviewable generated collection surfaces in Admin Studio.
- Provide standard CRUD UI, field inputs, validation banners, sortable tables, and search filters.
- Maintain strict CSRF token validation and session scoping.
- Enforce the Information Hierarchy: technical error codes, stack traces, and database internals are hidden from editors.

---

## Phase 3 — Editorial & Multi-Block Workflows

- First-class draft, review, scheduled publication, and archiving state machines.
- Visual revision diffs (side-by-side green/red changes) with point-in-time rollback.
- Structured rich-text and block editors (Quill, Editor.js) integrated with CSRF-protected live previews.
- Deep relational linkage with search pickers and modal creation dialogs.

**Exit Criteria:** Full editorial workflow verified end-to-end in News and Corporate edition fixtures.

---

## Phase 4 — Capability Kit & Plugin Ecosystem

- Versioned kit contracts, dependency graphs, migrations, permission scopes, and compatibility matrices.
- Official kits: SEO, Forms, Redirects, Full-Text Search, S3/CDN Media, Localization, Multi-Tenancy, Webhooks, Push Notifications.
- Sandboxed community kit specifications; untrusted dynamic Go code is never executed at runtime.

---

## Phase 5 — Frontend SDKs & Starter Presets

- Versioned TypeScript client, React Query / SWR hooks, strongly typed form and table components.
- Provider-agnostic design system adapters: Hydro Default, shadcn, headless, vanilla.
- Preset starters: `corporate`, `news`, `agency`, `saas`, `minimal`.
- Deterministic seeding, Playwright E2E tests, and deployment configurations for every starter.

---

## Phase 6 — Production & Upgrade Verification

- Automated S3/CDN, backup/restore, observability, and disaster recovery drills.
- SQLite and PostgreSQL capacity benchmarks; zero-downtime blue/green reload and rollback.
- Rehearsal of expand-migrate-contract schema migrations.
- Backwards-compatibility checks for SDKs, runtimes, and module boundaries.

---

## Definition of "Done"

HydroGo achieves full developer velocity parity when all three conditions are met:
1. Standard resource blueprints produce runnable Admin, API, seeds, and test suites in under 30 minutes.
2. Verified production baselines are measured and documented on real hardware.
3. Core, module, kit, and client edition upgrades pass automated compatibility guards without regressions.

---

## Platform-Factory Expansion

This architecture scales HydroGo from a CMS engine into a general-purpose Workspace Factory producing customized corporate platforms, SaaS backends, editorial publications, and internal portals.

### P0 — Core Bloat Audit & Boundary Integrity (Mandatory Prerequisite)
- Classify all internal packages as Core primitives, opt-in modules, kits, Edition contracts, or UI components.
- Audit dependencies for binary size, startup latency, memory footprint, and licensing terms.
- Guarantee that unused modules instantiate zero routes, migrations, assets, workers, goroutines, or database pools.
- Keep domain-specific business logic in Edition boundaries; never fork Core.

### P1 — Versioned Domain & Resource Contracts
Blueprint manifests support `collection`, `global`, `relation`, `value_object`, `computed`, `workflow`, `event`, `policy`, `job`, and `integration` declarations with schema diffing and breaking-change detection.

### P2 — Zero-Setup Identity & Policy Engine
Activating identity leverages the existing Core auth engine (sessions, RBAC, TOTP MFA, device management) with deny-by-default tenant-aware access policies.

### P3 — Secure Querying & API Parity
Client query strings parse into strongly typed AST filters with strict operator allowlists (`equals`, `contains`, `in`, `exists`, `and`, `or`, `sort`, `limit`). Query builders generate parameterized SQL queries with certified dialect parity.

### P4 — Globals, Validation & Relations
Single-record Globals (e.g. site settings) maintain transactional integrity. Validation constraints (`min`, `max`, `regex`, `unique`, `email`, `immutable`) execute across HTTP, service, and database layers from a single declaration.

### P5 — Workflows, Lifecycles & Event Outbox
Editorial lifecycle transitions generate transactional domain events written to the durable Outbox queue, guaranteeing at-least-once asynchronous worker processing with dead-letter retries.

### P6 — Business Admin Studio
Generated collection views feature relation search pickers, nested blocks, tabs, bulk actions, column customization, optimistic locking, and visual revision diffs with full WCAG 2.1 A/AA accessibility.

### P7 — Multi-Profile Deployment Spectrum
Clear separation between server-rendered Templ/HTMX, headless REST/OpenAPI, and rich application profiles, supporting high-throughput cloud daemons down to zero-ops shared hosting.
