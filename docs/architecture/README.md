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
| [0022-media](0022-provider-neutral-media-storage.md) | Provider-neutral media storage boundary | Accepted |
| [0023-design](0023-provider-agnostic-design-systems.md) | Provider-agnostic design-system adapters | Accepted |
| [0024-generated](0024-generated-compatibility-checks.md) | Generated artifact compatibility checks | Accepted |
| [0025-locale](0025-locale-presentation-formatting.md) | Locale-aware presentation formatting | Accepted |
| [0026-openapi](0026-openapi-compatibility-preflight.md) | OpenAPI compatibility preflight | Accepted |
| [0027-fields](0027-provider-neutral-field-registry.md) | Provider-neutral field registry | Accepted |
| [0028-adapters](0028-generated-frontend-adapters.md) | Optional generated frontend adapters | Accepted |
| [0029](0029-relation-field-rendering.md) | Request-scoped relation field rendering | Accepted |
| [0030](0030-admin-list-query-contract.md) | Bounded admin list query contract | Accepted |
| [0031](0031-editorial-workflow-contract.md) | Provider-neutral editorial workflow | Accepted |
| [0032](0032-admin-locale-selector.md) | Provider-neutral admin locale selector | Accepted |
| [0033](0033-admin-query-url-encoding.md) | Stable admin query URL encoding | Accepted |
| [0034](0034-admin-query-allowlists.md) | Module-owned admin query allowlists | Accepted |
| [0035](0035-admin-query-toolbar.md) | Shared admin query toolbar | Accepted |
| [0036](0036-optional-capability-kit-registry.md) | Optional capability kit registry | Accepted |
| [0037](0037-generator-kit-manifest.md) | Generator kit manifest | Accepted |
| [0038](0038-kit-aware-runtime-wiring.md) | Kit-aware generated runtime wiring | Accepted |
| [0039](0039-kit-registry-validation.md) | Fail-closed kit metadata validation | Accepted |
| [0040](0040-generated-kit-types.md) | Generated typed capability metadata | Accepted |
| [0041](0041-e2e-kit-diagnostics.md) | Capability kits in E2E diagnostics | Accepted |
| [0042](0042-admin-kit-badges.md) | Accessible admin capability badges | Accepted |
| [0043](0043-dashboard-kit-summary.md) | Admin dashboard capability summary | Accepted |
| [0044](0044-kit-set-validation.md) | Deterministic capability kit sets | Accepted |
| [0045](0045-admin-kit-summary-styles.md) | Admin capability summary styles | Accepted |
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
| [0048](0048-bounded-test-cli.md) | Bounded test command for the developer workflow | Accepted |
| [0049](0049-versioned-module-route-contract.md) | Versioned module route contract | Accepted |
| [0050](0050-module-dependency-graph-validation.md) | Module dependency graph validation | Accepted |
| [0051](0051-resource-create-cli-alias.md) | Resource-oriented CRUD scaffold command | Accepted |
| [0052](0052-migration-create-scaffold.md) | Reviewable migration scaffold command | Accepted |
| [0053](0053-semantic-ui-token-contract.md) | Semantic UI token contract | Accepted |
| [0054](0054-observability-cardinality-contract.md) | Bounded observability dimensions | Accepted |
| [0055](0055-incident-runbook-template.md) | Structured incident and rollback runbook template | Accepted |
| [0056](0056-canonical-module-filtering.md) | Canonical module names in migration filtering | Accepted |
| [0057](0057-release-checklist-gates.md) | Evidence-based release checklist | Accepted |
| [0058](0058-hydro-dev-entrypoint.md) | Local `hydro dev` entrypoint | Accepted |
| [0059](0059-dependency-triage-cadence.md) | Evidence-based dependency triage cadence | Accepted |
| [0060](0060-migration-metadata-lint.md) | Lint migration safety metadata before release | Accepted |
| [0061](0061-sdk-compatibility-assertions.md) | Compile-time assertions for extension contracts | Accepted |
| [0062](0062-slo-config-contract.md) | Keep SLO configuration target-neutral | Accepted |
| [0063](0063-request-actor-session-correlation.md) | Correlate request, actor, and session fields in logs | Accepted |
| [0064](0064-deterministic-alarm-catalog.md) | Keep the operational alarm catalog deterministic | Accepted |
| [0065](0065-prometheus-exporter-boundary.md) | Keep metrics export explicit and bounded | Accepted |
| [0066](0066-reproducible-release-evidence.md) | Require reproducible release evidence without claiming provenance | Accepted |
| [0112](0112-pages-locale-fallback-integration.md) | Integrate configured locale fallback into public page reads | Accepted |
| [0113](0113-news-locale-fallback-integration.md) | Integrate configured locale fallback into public news reads | Accepted |
| [0114](0114-generated-query-input-bounds.md) | Bound generated frontend query inputs | Accepted |
| [0115](0115-kit-descriptor-metadata.md) | Generate deterministic kit descriptors | Accepted |
| [0116](0116-generated-client-query-cancellation.md) | Make generated clients consume bounded queries | Accepted |
| [0117](0117-headless-client-timeout-and-error-bound.md) | Bound headless client waits and error bodies | Accepted |
| [0118](0118-frontend-manifest-check-command.md) | Expose standalone frontend manifest validation | Accepted |
| [0119](0119-bounded-frontend-manifest-input.md) | Bound frontend manifest inputs | Accepted |
| [0120](0120-versioned-taxonomy-contract.md) | Versioned taxonomy SDK contract | Accepted |
| [0121](0121-versioned-forms-contract.md) | Versioned forms SDK contract | Accepted |
| [0122](0122-versioned-editorial-workflow-contract.md) | Versioned editorial workflow contract | Accepted |
| [0123](0123-versioned-search-contract.md) | Versioned search provider contract | Accepted |
| [0124](0124-versioned-capability-kit-contract.md) | Versioned capability kit contract | Accepted |
| [0125](0125-versioned-analytics-consent-contract.md) | Versioned analytics consent contract | Accepted |
| [0126](0126-generated-frontend-contract-types.md) | Generated frontend contract types | Accepted |
| [0127](0127-admin-taxonomy-editor-primitive.md) | Admin taxonomy editor primitive | Accepted |
| [0067](0067-explicit-demo-seeding.md) | Require explicit confirmation for demo seeding | Accepted |
| [0068](0068-portable-content-allowlist.md) | Keep portable export explicitly allowlisted | Accepted |
| [0069](0069-optional-module-lifecycle-contracts.md) | Keep module lifecycle hooks optional and Core-orchestrated | Accepted |
| [0070](0070-signed-module-metadata-design.md) | Design signed module metadata without a marketplace | Accepted |
| [0071](0071-frontend-delivery-choice.md) | Choose frontend delivery by demonstrated boundary needs | Accepted |
| [0072](0072-public-list-indexes.md) | Index bounded public and admin list reads | Accepted |
| [0073](0073-module-application-boundary-guard.md) | Guard module application packages from concrete adapters | Accepted |
| [0074](0074-reusable-upload-abuse-corpus.md) | Keep upload abuse checks in a reusable corpus | Accepted |
| [0075](0075-audit-access-and-pii-boundary.md) | Keep audit and member PII behind explicit admin access | Accepted |
| [0076](0076-secret-rotation-revoke-contract.md) | Make secret rotation and emergency revoke explicit contracts | Accepted |
| [0077](0077-bounded-csp-report-endpoint.md) | Accept bounded CSP reports without persistence or CSRF state | Accepted |
| [0078](0078-versioned-frontend-manifest.md) | Publish a versioned frontend route and selector contract | Accepted |
| [0079](0079-frontend-state-conventions.md) | Standardize frontend state and error semantics | Accepted |
| [0080](0080-profile-pool-transaction-contract.md) | Verify capacity profile pools and transaction atomicity | Accepted |
| [0081](0081-ui-accessibility-regression-contract.md) | Guard shared shell accessibility semantics | Accepted |
| [0082](0082-bounded-list-query-contract.md) | Keep page and news list projections query-bounded | Accepted |
| [0083](0083-locale-direction-contract.md) | Derive safe RTL/LTR document direction from locale | Accepted |
| [0084](0084-repository-error-mapping-contract.md) | Keep repository uniqueness errors stable across SQL dialects | Accepted |
| [0085](0085-release-asset-hygiene.md) | Verify embedded vendor asset reachability before release | Accepted |
| [0086](0086-user-facing-error-redaction.md) | Keep browser transport errors generic | Accepted |
| [0087](0087-blank-target-link-hardening.md) | Harden admin blank-target links | Accepted |
| [0088](0088-inactive-module-route-isolation.md) | Isolate routes for inactive modules | Accepted |
| [0089](0089-route-conflict-validation.md) | Validate route conflicts before startup | Accepted |
| [0090](0090-configured-public-home-locale.md) | Derive public home locale from runtime configuration | Superseded |
| [0091](0091-sdk-runtime-artifact-distribution.md) | Public SDK and private runtime artifact distribution | Proposed |
| [0092](0092-signed-schema-compatibility-preflight.md) | Signed schema compatibility preflight | Proposed |
| [0093](0093-permission-contract-preflight.md) | Permission contract preflight | Proposed |
| [0094](0094-typed-frontend-query-adapters.md) | Typed framework-neutral frontend query adapters | Accepted |
| [0095](0095-provider-neutral-taxonomy-contract.md) | Provider-neutral taxonomy contract | Accepted |
| [0096](0096-provider-neutral-search-contract.md) | Provider-neutral search contract | Accepted |
| [0097](0097-generated-swr-adapter.md) | Generated SWR frontend adapter | Accepted |
| [0098](0098-provider-neutral-form-contract.md) | Provider-neutral form contract | Accepted |
| [0099](0099-kit-dependency-resolution.md) | Deterministic capability-kit dependency resolution | Accepted |
| [0100](0100-generated-form-schema.md) | Generated form schema artifact | Accepted |
| [0101](0101-frontend-starter-manifest.md) | Frontend starter manifest | Accepted |
| [0102](0102-generated-kit-dependency-artifacts.md) | Generated kit artifacts include resolved dependencies | Accepted |
| [0103](0103-generated-frontend-index.md) | Generated frontend namespace entrypoint | Accepted |
| [0104](0104-bounded-admin-bulk-contract.md) | Bounded provider-neutral Admin bulk contract | Accepted |
| [0105](0105-generated-bulk-request-adapter.md) | Generated bulk request adapter | Accepted |
| [0106](0106-opt-in-admin-bulk-list-surface.md) | Opt-in Admin bulk list surface | Accepted |
| [0107](0107-bounded-async-relation-contract.md) | Bounded async relation contract | Accepted |
| [0108](0108-locale-fallback-chain.md) | Deterministic bounded locale fallback chain | Accepted |
| [0109](0109-s3-cdn-public-url-boundary.md) | Explicit bounded S3/CDN public URL boundary | Accepted |
| [0110](0110-bounded-private-media-urls.md) | Bounded private media presigned URLs | Accepted |
| [0111](0111-protected-media-url-endpoint.md) | Protected private media URL endpoint | Accepted |
| [0128](0128-bounded-memory-search-provider.md) | Bounded local search provider for Compact/prototype profiles | Accepted |
| [0129](0129-news-editorial-transition-policy.md) | Transactional injectable News editorial transition policy | Accepted |
| [0130](0130-durable-local-search-index.md) | Bounded durable local search index | Accepted |
| [0131](0131-durable-analytics-consent-store.md) | Durable aggregate analytics with explicit consent | Accepted |
| [0132](0132-sdk-v1-support-policy.md) | SDK v1 support and deprecation policy | Accepted |
| [0133](0133-schema-driven-form-builder-primitive.md) | Schema-driven Form Builder primitive | Accepted |
| [0134](0134-durable-form-registry.md) | Durable bounded form registry | Accepted |
| [0135](0135-role-specific-workflow-policy.md) | Role-specific editorial transition policy | Accepted |
| [0136](0136-bounded-http-idempotency-contract.md) | Bounded HTTP idempotency contract | Proposed |
| [0137](0137-commercial-license-issuer-boundary.md) | Separate vendor-only license issuer from customer runtime | Accepted |
| [0138](0138-provider-neutral-oidc-boundary.md) | Provider-neutral OIDC boundary | Accepted |
| [0139](0139-mobile-api-auth-boundary.md) | Versioned mobile API authentication boundary | Accepted |
| [0140](0140-tenant-isolation-scope.md) | Multi-tenant scope is opt-in and explicit | Accepted |
| [0141](0141-ai-provider-and-data-boundary.md) | AI is opt-in and provider-neutral | Accepted |
| [0142](0142-distributed-profile-certification-boundary.md) | Distributed profile certification boundary | Accepted |
| [0150](0150-member-mobile-auth-boundary.md) | Member mobile authentication boundary | Accepted |
| [0151](0151-bff-server-side-request-policy.md) | BFF server-side request policy | Accepted (scoped) |
| [0152](0152-bff-server-side-session-adapter.md) | BFF server-side session adapter | Accepted (bounded single-instance reference) |
| [0153](0153-bff-shared-session-backend.md) | BFF shared Redis/Valkey session backend | Accepted (provider certification open) |
| [0154](0154-tenant-scope-primitive.md) | Explicit tenant scope primitive | Accepted (foundation only; shared mode disabled) |
| [0155](0155-tenant-scoped-cache-api.md) | Tenant-scoped cache API | Accepted (opt-in foundation; shared mode disabled) |
| [0156](0156-postgres-tenant-rls-binding.md) | Transaction-local PostgreSQL tenant binding | Accepted (foundation only; RLS policies open) |
| [0157](0157-admin-studio-extension-boundary.md) | Runtime Admin Studio extension boundary | Accepted (declarative external-surface foundation) |
| [0158](0158-bounded-outbox-polling.md) | Bounded outbox polling lifecycle | Accepted (worker-loop foundation; production soak open) |
| [0159](0159-sqlite-external-write-cache-monitor.md) | SQLite external-write cache monitor | Accepted (bounded safety net; integration and soak open) |
| [0143](0143-binary-only-customer-distribution.md) | Binary-only customer distribution | Accepted |
| [0144](0144-customer-package-source-exclusion.md) | Customer package source exclusion | Accepted |
| [0145](0145-hydro-run-development-entrypoint.md) | `hydro run` development entrypoint; `hydro dev` compatibility alias | Accepted |
| [0146](0146-backend-first-runtime-root.md) | Backend-first runtime root with optional demo frontend | Accepted |

Implementation order and exit gates are tracked in the [Roadmap](../ROADMAP.md).

## Checklist for every development

1. Identify the ADRs affected by the change.
2. Verify that it does not break the boundary between the Client Edition and Core contract.
3. Review the SQLite/PostgreSQL behavior and the impact on future adapters.
4. Evaluate cache invalidation and stale data risk.
5. Confirm that it does not create unnecessary services or unbounded resource consumption in the Compact profile.
6. Verify that old and new versions can coexist during blue/green deployment.
7. If the migration requires a maintenance window, explain this in the release metadata.
