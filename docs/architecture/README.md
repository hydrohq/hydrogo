# HydroGo Architecture Decision Log

This directory is the main source of HydroGo's long-lived architectural decisions. The implementation plan describes the order of work, while ADRs explain why the system is built this way.

Questions not yet accepted and proposed answers are tracked in the [Architecture Decision Backlog](DECISION_BACKLOG.md).

Coding risks and open proof gates are in the [Pre-implementation review](../PRE_IMPLEMENTATION_REVIEW.md), roadmap, and [production gate checklist](../validation/PRODUCTION-GATES.md); security flows are in the [Threat model](../security/THREAT_MODEL.md). **An accepted ADR alone is not proof of implementation or testing; the actual status of each ADR is evaluated together with the roadmap and validation reports.**

If a new development changes one of the following topics, the relevant ADR must be read and a new ADR written if necessary:

- The boundary between Core and customer code
- Database, cache, media, or job queue infrastructure
- Deployment, migration, and backward compatibility
- Security boundary or default
- Module contract and code generation
- Supported scale and resource profiles

## ADR statuses

- **Proposed:** Direction is determined; must be verified by implementation or benchmark.
- **Accepted:** New developments must comply with this decision.
- **Superseded:** Link to the replacing ADR is specified.
- **Superseded (partially):** Which provisions have changed and the scope that remains valid are clearly specified; the two documents are read together.
- **Rejected:** Evaluated but not selected.

An accepted decision is not silently changed. If new information invalidates the decision, the old ADR is kept, a new ADR is written, and the two documents are linked to each other.

## Decision list

| ADR | Decision | Status |
|---|---|---|
| [0001](0001-product-and-deployment-model.md) | Product and customer deployment model | Accepted |
| [0002](0002-database-portability.md) | SQLite, MySQL and PostgreSQL support | Superseded |
| [0003](0003-cache-architecture.md) | Multi-tier professional cache | Partially superseded: 0011 |
| [0004](0004-scaling-profiles.md) | Numerical resource profiles | Superseded |
| [0005](0005-zero-downtime-deployment.md) | Blue/green and zero-downtime release transition | Partially superseded: 0011 |
| [0006](0006-extensible-database-adapters.md) | Initial version SQLite/PostgreSQL and extensible DB adapters | Accepted |
| [0007](0007-capacity-profiles.md) | Capacity profiles without numerical hardware promise | Accepted |
| [0008](0008-module-and-extension-contract.md) | Initial module/extension contract | Superseded: 0010 |
| [0009](0009-site-edition-and-delivery-scope.md) | Site Edition and phased delivery scope | Accepted; planned |
| [0010](0010-sdk-lifecycle-and-edition-manifest.md) | SDK maturity, module lifecycle and manifest | Accepted; planned |
| [0011](0011-consistency-upgrade-and-recovery.md) | Cache/transaction consistency, upgrade and recovery | Accepted; planned |
| [0012](0012-content-media-and-portability.md) | Content, media and data portability | Partially superseded: 0014; planned |
| [0013](0013-identity-and-security-boundaries.md) | Identity, authority and security boundaries | Partially superseded: 0014; planned |
| [0014](0014-publication-routes-and-erasure-recovery.md) | Route/SEO during publish and erasure history independent of restore | Accepted; planned |
| [0015](0015-versioned-http-api.md) | Versioned HTTP API routes and backward alias | Accepted |
| [0016](0016-structured-page-blocks-and-theme-slots.md) | Structured page blocks and theme slots | Accepted |
| [0017](0017-commerce-consistency-and-payments.md) | Commerce consistency and payment providers | Accepted |
| [0018](0018-filesystem-theme-bundles.md) | File-based theme bundles and secure slot renderer | Accepted |
| [0019](0019-external-webhook-delivery.md) | Signed external webhook delivery | Accepted |
| [0020](0020-headless-first-admin-studio-and-frontend-templates.md) | Headless-first Admin Studio and frontend template boundary | Proposed |
| [0021](0021-headless-browser-session-boundary.md) | Headless browser session and cross-origin boundary | Proposed |
| [0022](0022-consent-based-product-analytics-and-legal-content.md) | Consent-based product analytics and manageable legal content | Proposed |
| [0023](0023-metadata-driven-e2e.md) | Metadata-driven Admin and Frontend E2E | Proposed |
| [0024](0024-optional-redis-rate-limiting-and-queue-boundary.md) | Optional Redis rate limiting and SQL queue boundary | Accepted |
| [0025](0025-separate-frontend-members.md) | Separate frontend member identity | Accepted for MVP |
| [0026](0026-bounded-media-derivatives.md) | Bounded media derivatives | Accepted |
| [0028](0028-core-isolation-and-licensing.md) | Core isolation and commercial licensing | Accepted |
| [0029](0029-commerce-customer-capability-boundary.md) | Commerce customer capability and public catalog boundary | Accepted |
| [0030](0030-client-address-trust-boundary.md) | Client IP derivation and proxy trust boundary | Accepted |
| [0031](0031-commerce-reservation-expiry.md) | Bounded pending-order stock reservation expiry | Accepted |
| [0032](0032-payment-provider-transport-safety.md) | Bounded, redirect-safe payment provider transport | Accepted |
| [0033](0033-runtime-module-resource-gating.md) | Runtime resources gated by active modules | Accepted |
| [0034](0034-refund-reconciliation-state.md) | Idempotent refund reconciliation state | Accepted |
| [0035](0035-secure-payment-return-url-validation.md) | Secure payment return URL validation | Accepted |
| [0036](0036-bounded-public-catalog-projections.md) | Bounded public catalog projections | Accepted |
| [0037](0037-strict-modular-monolith-boundaries.md) | Strict modular monolith boundaries | Accepted |
| [0038](0038-member-order-capability-contract.md) | Member order capability contract | Accepted |
| [0039](0039-conditional-cross-module-migrations.md) | Conditional cross-module migration scopes | Accepted |
| [0040](0040-conditional-headless-schema.md) | Conditional headless schema surface | Accepted |
| [0041](0041-admin-visual-theme-modes.md) | Admin visual theme modes | Accepted |
| [0042](0042-referer-redirect-boundary.md) | Referer redirect boundary for admin mutations | Accepted |
| [0043](0043-path-segment-redirect-escaping.md) | Escape dynamic redirect path segments | Accepted |
| [0044](0044-sanitized-theme-content-contract.md) | Sanitized content contract for example themes | Accepted |
| [0045](0045-task-oriented-module-cli.md) | Task-oriented module creation CLI | Accepted |
| [0046](0046-offline-configuration-validation.md) | Offline configuration validation command | Accepted |
| [0047](0047-module-authoring-scaffold.md) | Reviewable module authoring scaffold | Accepted |

Implementation order and exit gates are tracked in the [Roadmap](../ROADMAP.md).

## Checklist for every development

1. Identify the ADRs affected by the change.
2. Verify that it does not break the boundary between the Client Edition and Core contract.
3. Review the SQLite/PostgreSQL behavior and the impact on future adapters.
4. Evaluate cache invalidation and stale data risk.
5. Confirm that it does not create unnecessary services or unbounded resource consumption in the Compact profile.
6. Verify that old and new versions can coexist during blue/green deployment.
7. If the migration requires a maintenance window, explain this in the release metadata.
