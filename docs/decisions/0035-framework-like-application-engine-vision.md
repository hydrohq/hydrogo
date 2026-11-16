# 0035: Realigning HydroGo as a Framework-like Go CMS & Application Engine

- Status: Accepted
- Date: 2026-11-16

## Context

In ADR 0001 (`0001-cms-focus.md`), HydroGo was refocused away from an unwieldy
code generator into a "single-binary headless Go CMS". While this helped trim
unwieldy extensions, its strict rule ("Go serves JSON, not markup; no server-side
HTML rendering; never render a public website") clashed with the true real-world
needs of the platform.

In practice, modern engines like PocketBase, PayloadCMS, and Supabase succeed because
they are **Framework-like Application Engines**: they supply robust headless APIs and
embedded administration, while offering turnkey frontend starters (`sveltekit-starter`,
`boutique-starter`, `commerce-starter`, `crm-starter`) and out-of-the-box presentation
capabilities tailored to each project's business purpose.

Constraining the project to a purist "headless-only" definition created architectural
friction: developers built rich multi-tenant storefront archetypes and SSR templates
in `internal/storefront` while having to circumvent the declared headless constraint.

## Decision

1. **Constitutional Alignment**:
   - Officially recognize HydroGo as a **Single-Binary Framework-like Go CMS & Application Engine**.
   - HydroGo delivers:
     - A high-performance REST API and schema-driven content engine (`internal/contenttypes`, `internal/collections`).
     - An embedded, reactive Svelte 5 administration workspace (`admin/`).
     - Purpose-driven project scaffolding (`hydro new --preset <name> --starter <frontend>`) delivering turn-key frontends out of the box.
     - Multi-tenant boutique storefront archetypes and hybrid SSR capabilities for fast landing, showcase, and digital commerce.

2. **First-Class Starter & Preset Integration**:
   - Frontends (`examples/sveltekit-starter`, `examples/boutique-starter`, etc.) are recognized as first-class components of the HydroGo ecosystem, not external afterthoughts.
   - When scaffolding a new project, the preset and starter work hand-in-glove based on the intended business archetype (Boutique, News, SaaS, Corporate, CRM, ERP-Lite).

3. **Core Decoupling**:
   - Storefront and archetype routing in Go serves as a robust hybrid fallback and edge presenter, while hardcoded language paths (e.g. Turkish URL slugs) are structured to remain configurable.

## Consequences

- Resolves the philosophical split-brain across documentation and codebase.
- `AGENTS.md` is updated to reflect this unified framework-like CMS identity.
- Development velocity improves as frontend starters, storefront archetypes, and admin schemas operate under a shared, consistent architecture.
