# Decision records

| # | Decision | Status |
|---|---|---|
| [0001](0001-cms-focus.md) | Refocus HydroGo as a CMS | Accepted |
| [0002](0002-content-types.md) | One way to define a content type | Accepted |
| [0003](0003-social-sign-in.md) | Social sign-in for both admin staff and members | Accepted |
| [0004](0004-theme-appearance-css.md) | Theme contract carries the admin's Appearance CSS | Superseded by 0005 |
| [0005](0005-admin-svelte-spa.md) | Headless core, admin panel rebuilt as a Svelte SPA | Accepted |
| [0006](0006-freeze-legacy-peripherals-and-view-pipeline.md) | Freeze legacy peripherals and native view pipeline | Accepted |
| [0007](0007-template-cms-distribution.md) | Distribute HydroGo as a template CMS | Accepted |
| [0008](0008-business-application-capabilities.md) | Business-application capabilities in core | Proposed |
| [0009](0009-core-owned-event-outbox-and-migration-baseline.md) | Core-owned event outbox and a migration baseline for new databases | Accepted |
| [0010](0010-generic-reports-and-jobs.md) | Generic reports and job visibility in core, not per project | Accepted (Slice A); Proposed (B, C) |
| [0011](0011-generated-role-permission-tests.md) | Generate per-role permission tests from content types | Accepted (Phases 1-2); Proposed (Phase 3) |
| [0012](0012-commerce-catalog-as-content-types.md) | Commerce catalog as content types, not a new engine feature | Accepted (Phase 1 of 4) |
| [0013](0013-commerce-cart-checkout-module.md) | `modules/commerce` for stock, cart, checkout and orders | Accepted (Phase 2 of 4) |
| [0014](0014-commerce-paytr-provider.md) | PayTR as a `commerce.PaymentProvider` | Accepted (Phase 3 of 4) |
| [0015](0015-commerce-starter.md) | `examples/commerce-starter` | Accepted (Phase 4 of 4) |
| [0016](0016-commerce-starter-catalog-sections.md) | Commerce starter catalog sections (category pages, search, galleries) | Accepted |
| [0017](0017-commerce-tax-shipping-discounts-currencies.md) | Tax, shipping, discount codes and display currencies | Accepted |
| [0018](0018-commerce-reconciliation.md) | Commerce reconciliation and drift alerting | Accepted |
| [0019](0019-commerce-order-notifications-and-member-login.md) | Order notifications and storefront member login | Accepted |
| [0020](0020-commerce-shipping-provider.md) | Generic `ShippingProvider`, Yurtiçi Kargo first | Accepted |
| [0021](0021-commerce-admin-orders-and-payment-scope.md) | Admin order screen, order district, and parking payment/carrier verification | Accepted |
| [0022](0022-audit-remediation.md) | 2026-10-24 audit remediation | Accepted |
| [0023](0023-frontend-toolchain-bump-and-sveltekit-3-blocker.md) | Admin toolchain bump; SvelteKit 3 blocked by an env-folding regression | Accepted |
| [0024](0024-news-taxonomy-tenant-isolation.md) | News taxonomy had no tenant isolation at all | Accepted |
| [0025](0025-sveltekit-cve-mitigation-without-the-major-bump.md) | Closing the two SvelteKit CVEs without the blocked version bump | Accepted |
| [0026](0026-sveltekit-starter-static-adapter.md) | `sveltekit-starter` converted to a static SPA (adapter-static) | Accepted |
| [0027](0027-realtime-baas-and-row-level-access-rules.md) | Realtime BaaS (PocketBase Parity) and Row-Level Access Rules | Accepted |
| [0028](0028-ai-agent-control-plane-and-mcp.md) | AI-Agent Control Plane & Model Context Protocol (MCP) Architecture | Accepted |
| [0029](0029-platform-orchestrator-and-resource-governor.md) | Platform Orchestrator & Resource Governor | Accepted |
| [0030](0030-ai-workspace-copilot-and-boutique-factory.md) | AI Workspace Copilot & 60-Second Boutique Store Factory | Accepted |
| [0031](0031-ai-generation-and-prompt-registry.md) | AI Generation Engine, Prompt Registry, and Section-Based Storefront | Accepted |
| [0032](0032-fail-closed-rls-and-zero-dependency-memory-engine.md) | Fail-Closed PostgreSQL RLS, Zero-Dependency In-Memory Engine, and Storefront Resilience | Accepted |

Write a record only when a change moves a module boundary, changes a public
contract (routes, API, config, migrations), or reverses a decision listed here.
Keep it under a page: context, decision, consequences. Older records live in
`docs/archive/architecture/` for history only.
