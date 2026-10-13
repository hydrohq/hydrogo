# HydroGo Implementation Status & Operational Guide

This document distinguishes between planned items and verified delivered capabilities. An item marked as "completed" here indicates that code resides in the repository and passes local verification suites; it does not imply external production certification.

For the audit and staged delivery plan regarding Blueprint Admin Studio client experience, Super Admin boundaries, automatic activation, and verification gaps, see:
[`docs/planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md`](planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md).

2026-10-10 Content and Form localization follow-up: list action buttons (`Edit`,
`View`, `Copy live URL`, `Import CSV / Excel`)
and table action headers (`Actions`) in Pages and News modules, as well as
`RenderCreateForm` widgets (media picker, item list, relation select, content language navigation),
are bound to the request's `hydro_admin_locale` cookie preference. Comprehensive unit and transport
tests added in `internal/adminstudio`, `modules/pages`, and `modules/news`.

2026-10-10 native clean scaffold Blueprint benchmark follow-up: clean scaffold (`new`),
Blueprint generation (`resource create`), and project contract check (`project check`) were
benchmarked across 3 clean iterations on temporary roots. Warm cache median total time: 0.780s
(min: 0.774s, max: 0.780s), producing 70 total files. Full composition and licensed runtime execution
remain documented as host/CI gates. Evidence:
`native-clean-scaffold-blueprint-2026-10-10.md`.

2026-10-10 generated Admin conflict form state follow-up: handled 409 Conflict in generated
admin HTTP endpoints by re-rendering the full form with `ValidationSummary` alert banner
(`admin.generated.error.conflict`) and preserving submitted form values (preventing lost editor input
on concurrent edit collisions). Bilingual (TR/EN) conflict tests passed. Evidence:
`generated-admin-conflict-form-state-2026-10-10.md`.

2026-10-10 Pages & News denied mutations storage isolation follow-up: `spyPageRepository`
and `spyNewsRepository` prove that denied mutations (viewer, restricted editor, missing/invalid CSRF)
to pages and news endpoints return 403 and invoke zero storage/persistence operations. Evidence:
`page-news-denied-mutations-storage-isolation-2026-10-10.md`.

2026-10-09 Admin role-matrix follow-up: denied Editor/Viewer account role,
permission, and disable mutations are followed by a fresh Admin read that
checks the target role, permission override, and active state remain unchanged.
The five-test Chromium Admin-management file passed on SQLite and PostgreSQL
18 using a `NOSUPERUSER NOBYPASSRLS` application role and separate
`BYPASSRLS` backup operator. This verifies the three account mutations only;
the complete Admin route/mutation matrix remains open. Evidence:
`admin-denied-account-mutations-persistence-2026-10-09.md`.

2026-10-09 generated Admin UX follow-up: the native Project Edition Chromium
CRUD scenario persists 26 generated records and verifies the 25-row list
boundary, page-two remainder, and return navigation on SQLite and PostgreSQL
18. This covers the normal two-page case for one collection; other collection
implementations and maximum-page behavior remain open. Evidence:
`generated-admin-pagination-browser-2026-10-09.md`.

2026-10-09 generated Admin failure-feedback follow-up: the independent Edition
Chromium flow injects a 503 on a valid create, checks localized retry
feedback, retained input, and absence of persistence, then retries
successfully. Editor/Viewer and media browser journeys passed 2/2 on SQLite
and PostgreSQL 18. This is one create failure path in a test fixture; other
API failure points and production outage behavior remain open. Evidence:
`generated-admin-service-error-browser-2026-10-09.md`.

2026-10-09 independent Edition activation follow-up: a third resource generated
in an independent Edition copy appeared in the existing aggregate Admin/API
helpers without per-resource composition edits. Chromium create/readback passed
on SQLite and PostgreSQL 18 after explicit Edition migration application. The
flags-based resource command now chooses a usable string/text title field;
clean scaffold to licensed runtime remains open. Evidence:
`blueprint-independent-edition-autoactivation-2026-10-09.md`.

2026-10-09 Admin security role follow-up: valid-CSRF Editor/Viewer IP-ban add,
unban, and token-lifetime updates are denied; an Admin re-read proves the
address was not added and both lifetimes remain unchanged. The complete
Admin-management browser file passed on SQLite and PostgreSQL 18 with the
documented separate backup role boundary. Other system mutation persistence
cells remain open. Evidence:
`admin-security-denied-mutation-persistence-2026-10-09.md`.

2026-10-09 Admin CSRF follow-up: Editor and Viewer requests to every same-origin
POST action discovered on ten privileged Admin pages are rejected with 403
when the CSRF token is valid, missing, or mismatched. Admin sessions also
receive 403 for missing/mismatched tokens at each target. The full five-test
Admin-management Chromium file passed on SQLite and PostgreSQL 18.6. This
covers rendered targets only; valid-CSRF Admin writes, API-only/module routes,
and per-target persisted-state checks remain open. Evidence:
`admin-rendered-post-csrf-matrix-2026-10-09.md`.

2026-10-09 generated Admin CSRF follow-up: all nine generated collection
mutation handlers now separately reject missing and mismatched tokens before
resource API or Edition callback dispatch. Focused and full `internal/adminstudio`
tests plus package vet passed. This is handler contract coverage; independent
Edition browser persistence and database adapter evidence remains scoped to
the separately documented CRUD/editorial journeys. Evidence:
`generated-admin-validation-feedback-2026-10-09.md`.

2026-10-09 public Admin accessibility follow-up: invite-accepted and
password-reset confirmation states passed WCAG 2.1 A/AA in Turkish/English
and light/dark after fixing three light-mode contrast failures. The full
Admin-management Chromium file passed 6/6 on SQLite and PostgreSQL 18.6;
`go test ./internal/ui -count=1` and the Impeccable detector passed. Other
public login error states and production deployment routes remain open.
Evidence: `admin-login-confirmation-accessibility-2026-10-09.md`.

2026-10-09 Admin login error follow-up: real invalid-credential submissions
receive the localized generic 401 alert and passed WCAG 2.1 A/AA in English/
Turkish and light/dark (4/4). A light-mode contrast failure at 3.08:1 was
fixed with a darker error token. The complete Admin-management browser file
passed 7/7 on SQLite and PostgreSQL 18.6; `go test ./internal/ui -count=1`
and the Impeccable detector passed. MFA/rate-limit states remain open.
Evidence: `admin-login-error-accessibility-2026-10-09.md`.

2026-10-09 Admin Studio follow-up: the public generated Studio renderer now
supports the Edition's Viewer presentation role. The Edition still owns route,
resource, tenant, and API authorization; read-only Viewer composition is covered
by SQLite contract tests. UI contract tests confirm content navigation remains
visible while privileged system links are hidden. The real Chromium session
remains an external listener-capable gate; see ADR-0365 and the audit plan
evidence.

2026-10-09 permission-matrix follow-up: media Admin management routes now have
valid-CSRF and invalid-CSRF denial assertions for Editor and Viewer defaults
and an explicit Admin `media.manage` deny override. The cleanup route keeps its
separate Admin-only boundary. See `admin-permission-matrix-2026-10-08.md`.

2026-10-09 Admin Studio verification: the complete root Go suite passed with
`go test ./... -count=1 -p 1 -timeout=5m`, and `go vet ./...` passed. The
verified run used a disk-backed Go cache and a temp directory outside the
module. PostgreSQL cases needing `HYDRO_TEST_POSTGRES_URL` were skipped because
the variable is unset; the external Edition Chromium journeys remain
unexecuted because this runner cannot bind loopback. Do not treat these local
results as PostgreSQL or browser certification; details and remaining gates
are in the Admin Studio audit plan.

Generated required text validation now rejects whitespace-only values and the
external Edition proves its 422 field feedback and no-persistence behavior.
The clean CLI preparation path (scaffold, one Blueprint resource, project
check) was measured in three warm-binary runs at 15.1 ms, 18.3 ms, and 15.6 ms
median per command. It emits no `go.mod` or runnable Edition server, so native
first CRUD remains unmeasured; this timing is not presented as equivalent.
See the Admin Studio audit plan and
`docs/validation/blueprint-clean-scaffold-cli-measurement-2026-10-09.md`.

Latest Admin Studio regression after the mobile generated-editor and E2E
fixture-reset changes: root `go test ./... -count=1 -p 1 -timeout=8m`,
`go vet ./...`, moduleguard, migrationguard (110 migrations), and four Edition
importguards passed. `HYDRO_TEST_POSTGRES_URL` was unset for the root Go run;
PostgreSQL claims are limited to the separately recorded isolated browser
fixture runs, not the root DB contract suite.

## Completed and Verified Capabilities

| Area | Status | Usage / Verification | Evidence |
| --- | --- | --- | --- |
| Consumer/member mobile API | Completed (local) | Dedicated `members.Member` bearer session with register/login/refresh, device revocation, profile, cart, checkout, order history, and ownership-scoped payment status | ADR-0150, `R66-release-gate-manifest-2026-09-19`, `R69-member-order-detail-payment-status-2026-09-19`, `docs/api/member-mobile-v1.openapi.json`, `docs/api/member-commerce-v1.openapi.json` |
| Public project command space | Completed | `hydro project new`, `hydro project check` (dependency-free verification of generated resources when present) | `R5-sdk-project-namespace`, `R5-sdk-project-check`, `R6-project-resource-preflight` |
| Payload-compatible CLI flow | Completed | `check`, `module new`, `resource new`, `resource check`, and `api diff`; legacy aliases preserved | `R5-cli-new-aliases` |
| Signed schema/migration diff | Completed | Compares release ledger checksums; deletions/modifications fail closed | `R5-schema-diff` |
| Permission contract preflight | Completed | Safe, collision-free permission identifiers and `permissions check` | `R5-permissions-check` |
| Generated SDK guide | Completed | Emits `docs/SDK.md` in newly scaffolded projects | `R5-generated-sdk-guide` |
| Runtime artifact verification | Completed | `hydro runtime verify [--root PATH]` | `R5-runtime-artifact-verification` |
| Signed manifest artifact hygiene | Completed | Fail-closed checks for duplicate entries, paths, sizes, and checksums | `R5-release-manifest-artifact-hygiene` |
| Platform targets & artifact resolution | Completed | Supports `linux-amd64`, `windows-arm64`, and other target platforms | `R5-sdk-platform-targets`, `R5-sdk-artifact-resolution` |
| Secure runtime cache paths | Completed | Deterministic cache directory names keyed by product, version, and target | `R5-sdk-runtime-cache` |
| Cache listing and garbage collection | Completed | Version retention allowlist + dry-run + rejection of symlinks | `R5-sdk-cache-gc` |
| Atomic artifact installation | Completed | Checksum and size verification + atomic filesystem rename | `R5-sdk-atomic-install` |
| Download URLs & bounded downloads | Completed | Enforces HTTPS, redirects disabled, and bounded download sizes | `R5-sdk-download-url-policy`, `R5-sdk-bounded-download` |
| Grant verification & download binding | Completed | Validates project, core, SDK, expiration, and URL constraints | `R5-sdk-artifact-grant`, `R5-sdk-grant-download-integration`, `R5-sdk-grant-parser` |
| Broker grant client | Completed (local) | Bounded POST, coordinates/features preflight, token headers, redirect/status/response validation, and single-call lock verification | `R5-sdk-broker-client` |
| Typed frontend query adapter | Partially completed (local) | Framework-neutral typed query options and stable query key helpers in generated `react.ts`; generated API client enforces bounded queries and `AbortSignal` | `R5-typed-frontend-query-adapter`, `R5-generated-client-query-cancellation` |
| Taxonomy contract | Partially completed (local) | Provider-neutral contract validating multilingual category/tag structures, parent cyclic checks, and related-content references | `R5-taxonomy-contract` |
| Search contract | Partially completed (local) | Bounded query/locale/pagination and deterministic result contracts; provider/index selection open | `R5-search-contract` |
| Generated SWR adapter | Partially completed (local) | Dependency-free typed key/fetcher generator; browser integration open; namespaced `index.ts` entry point emitted for all adapters | `R5-generated-swr-adapter`, `R5-generated-frontend-index` |
| Generated frontend query bounds | Completed (local) | Generated query adapters enforce page/page-size/sort/filter bounds before request URL generation; browser/provider acceptance open | `R5-generated-query-input-bounds` |
| Form contract | Partially completed (local) | Bounded metadata and submission normalization; inbox/consent/delivery remain module-owned | `R5-form-contract` |
| Kit dependency resolution | Partially completed (local) | Allowlisted dependency expansion, validated registry constructor, and deterministic kit sets; generated admin/SDK/runtime artifacts include resolved dependencies and typed optional descriptors; implementation loading/compatibility matrix remain module-owned | `R5-kit-dependency-resolution`, `R5-generated-kit-dependency-artifacts`, `R5-kit-descriptor-metadata`, `R6-validated-kit-registry` |
| Generated form schema | Partially completed (local) | Deterministic `form.schema.json` from collection spec; business validation/inbox remain module-owned | `R5-generated-form-schema` |
| Frontend starter manifest | Partially completed (local) | Credential-free manifest declaring preset, presentation, features, and design-system options | `R5-frontend-starter-manifest` |
| Frontend manifest CLI check | Completed (local) | `hydro frontend check --manifest PATH [--json]` executes identical fail-closed route validation without project generation | `R5-frontend-manifest-check-command` |
| Frontend manifest input bounds | Completed (local) | Rejects manifest files exceeding 1 MiB, non-regular files, or unbounded route name/path/test-id lengths | `R5-bounded-frontend-manifest-input` |
| Generated resource self-description | Completed (local) | Every generated collection contains deterministic `spec.json` and artifact checksums; `hydro resource check --root` discovers specs and validates entire scaffold dependency-free and fail closed | `R6-generated-resource-discovery`, `R6-generated-artifact-integrity` |
| Generated OpenAPI query/error contract | Completed (local) | OpenAPI outputs declare bounded list query parameters and standard `HydroAPIError` responses; `hydro validate api` catches parameter/body removals and optional-to-required breaking changes | `R6-generated-openapi-contract`, `ADR-0026` |
| Versioned global resource contract | Completed (local) | Blueprint `type: global` emits singleton resource metadata, global ID normalization, and admin manifest/API contracts | `ADR-0224`, `305072e1`, `c5f702df` |
| Generated global singleton transport | Completed (local) | Empty route GET on global resources resolves to singleton record via stable `global` ID; collection list surface omitted | `ADR-0224`, `0338bf24`, `c57ce5d7` |
| Bounded public relation plan | Completed (local) | Bounds cardinality, depth, edge count, and item budgets for relation resolvers/eager loaders via provider-neutral contracts; authorization and tenant policy remain Edition-owned | `ADR-0230` |
| DX iteration benchmark refresh | Completed (local measurement) | Scaffolding, resource generation, and compatibility acceptance median of 1.706 s across 3 clean Windows runs; parity claims withheld pending matching Payload runs | `docs/validation/dx-iteration-2026-09-22-deep-query.json` |
| Two-resource golden-path baseline | Completed (HydroGo + Payload smoke) | 3 runs each across Testimonial and independent Article fixtures; medians: 1.651 s / 1.546 s. Payload 3.90.1 SQLite harness measured for build, cold/warm admin, and warm API smoke; authenticated CRUD, p95/p99, and capacity parity remain open | `docs/validation/R102-hydrogo-golden-path-baseline-2026-09-22.md` |
| Full-suite/static gate refresh | Completed (local Windows) | Full `go test`, `go vet`, HTML comment lint, and ADR drift checks passed | `docs/validation/R103-full-suite-2026-09-22.md` |
| Generated global singleton DB invariant | Completed (contract level) | Generated migrations for `type: global` enforce `CHECK (id = 'global')` to reject secondary rows; operator migrations required for SQLite/PostgreSQL | `ADR-0238`, generator global contract test |
| Production gate runner PowerShell compatibility | Completed (local script) | Resolved Windows PowerShell 5 parser error; preflight fails closed when URL is missing | `docs/validation/R104-production-gate-script-2026-09-22.md` |
| WSL PostgreSQL/Redis production-gate refresh | Completed (local services) | Full/shuffled/race, reproducible build, portable benchmark, and Redis atomic limiter gates passed; real restore/proxy/key/capacity drills open | `docs/validation/R105-wsl-production-gates-2026-09-22.md` |
| WSL restore/Redis disposable drills | Completed (local services) | Disposable PostgreSQL backup/restore + contracts and temporary Redis atomic/race drills passed; external production proxy/key/capacity drills open | `docs/validation/R106-wsl-restore-redis-drills-2026-09-22.md` |
| Generated global Admin surface | Completed (contract level) | Global resource list template omits tables and pagination; redirects to singleton form surface, verified via generated tests | `ADR-0239`, generator global admin test |
| Capacity profile baseline refresh | Completed (local smoke) | Measured Compact/Standard/Distributed profiles across 100 warm-up + 20 sample iterations; peak working set: 92.2 / 94.8 / 101.5 MiB; does not constitute customer SLA | `docs/validation/R107-capacity-profile-baseline-2026-09-22.md` |
| Global contract static refresh | Completed (local) | Generator global tests, ADR drift check (261 ADRs), and HTML comment lint passed | `docs/validation/R108-static-contract-refresh-2026-09-22.md` |
| Post-push security revalidation | Completed (source level) | Pushed global/gate changes compared against prior audit; zero new source-level findings; provider and penetration gates open | `docs/validation/R109-security-post-push-revalidation-2026-09-22.md` |
| Core over-engineering audit | Completed (audit) | Repository-wide analysis revealed zero safe deletions or direct dependency replacements; deprecated v0 Module removal deferred to compatibility window | `docs/validation/R110-ponytail-audit-2026-09-22.md` |
| Core footprint / P0 baseline | Partial (audit; P0 open) | Windows amd64 universal binary measured at 36.98 MiB; sample Editions measured in CI against 15.05 MiB / 89 non-stdlib package budget; `profile explain/check` not yet coupled to real Edition composition/link graphs, and full Edition/inactive resource gates open | `ADR-0247`, `R120-core-footprint-baseline-2026-09-22.md`, `R121-profile-cli-contract-2026-09-22.md`, `R122-edition-footprint-gate-2026-09-22.md` |
| Bounded Admin saved-view contract | Completed (local) | Validates column, filter, and sort metadata for user views against field/operator allowlists and size budgets; execution policy remains Edition-owned | `ADR-0231` |
| Bounded offline sync contract | Completed (local) | Validates cursor, idempotent commands, payload, and batch budgets for field/offline Editions; conflict and attachment policies remain Edition-owned | `ADR-0232` |
| Public tenant scope contract | Completed (local) | Fail-closed tenant context and cache key contract for external Edition boundaries; authenticated resolver and DB RLS/predicate wiring remain Edition-owned | `ADR-0233` |
| Generated Admin view metadata | Completed (local) | Generates deterministic `admin/views.json` default view per resource, verified against compatibility lock; user persistence remains Edition-owned | `ADR-0234` |
| Versioned field validation binding | Completed (local) | Blueprint validation metadata wired into generated `validation.go` and executed by service Create/Update | `ADR-0226`, `110833a0`, `b985bf7b` |
| Client Edition lifecycle hook scaffold | Completed (local) | Scaffolds Before/After create/update/delete skeletons without overwriting existing custom hooks; routes side effects to outbox | `ADR-0227`, `5a98918d` |
| Public bounded resource query contract | Completed (local) | `extension/v1` parser enforces field/operator allowlists and query budgets importable by Client Editions | `ADR-0228`, `c67bdf71` |
| Generated bounded collection list transport | Completed (local) | Generated repository, service, and HTTP transport provide collection GET with page/offset limits and stable sort allowlists | `ADR-0229`, `00af1350`, `c3123829` |
| Generated route composition helper | Completed (local) | Generates policy-neutral `Register<Name>Route` in `runtime.go` per resource; tenant-scoped resources require resolvers, while migration/worker/middleware remain in Edition | `ADR-0240`, generator contract test |
| Resource auth composition flag | Completed (local) | Blueprint `auth: true` is propagated to Spec/module metadata, emitting an authenticated Identity/Session/RBAC wiring checklist; auth system and grants are never generated automatically | `ADR-0242`, blueprint loader tests |
| Generated auth route guard | Completed (local) | `auth: true` generated route registration fails closed without an Edition checker, returning 401 to anonymous requests; Identity/Session/RBAC implementations remain Edition-owned | `ADR-0243`, generator auth guard test |
| Generated bounded workflow contract | Completed (local) | Blueprint workflow graphs emit `workflow.json` with a 32 state/transition bound and known-state validation; transactions, role policies, schedules, and outbox execution remain module/Edition responsibilities | `ADR-0244`, blueprint workflow tests |
| Blueprint bounded relation metadata & many-valued storage | Completed (local; live PostgreSQL certification open) | Multi-valued relation fields persist strictly to normalized tenant-aware join tables; item-bound replacement executes atomically within its own or an enclosing Edition transaction; target authorization, eager loading, and cascade rules remain Edition/provider responsibilities | `ADR-0245`, `ADR-0246`, generator and SQLite Edition tests |
| Generated deep-query binding | Completed (local) | Public query AST parsed from generated transport; applies equals, contains, in, exists, and range filters with field allowlists and bound SQL | `ADR-0228`, `ADR-0229`, `c28d2746` |
| Bounded HTTP idempotency pilot | Partially completed (local) | Durable replay and conflict contracts on SQLite/PostgreSQL for Site form submissions, generated TypeScript mutation helpers, bounded key and request fingerprint conflict protection for Commerce checkout, and Stripe payment-intent provider key forwarding; iyzico/PayTR provider certifications open | `R20-form-idempotency`, `R21-generated-idempotency-client`, `R25-commerce-idempotency-key-validation`, `R26-stripe-payment-intent-idempotency`, `R27-commerce-order-fingerprint`, `ADR-0136` |
| Site inactive-module construction guard | Completed (local) | Site service, forms binding, limiter, and HTTP handlers instantiated strictly when Site module is enabled; preset route and migration acceptance remain separate evidence | `R22-site-module-zero-construction`, `R17-wsl-preset-module-isolation` |
| Pages cache inactive-module guard | Completed (local) | In-process and Redis page caches omitted in profiles where Pages is inactive; metrics provider remains nil-safe | `R23-pages-cache-zero-construction`, `R17-wsl-preset-module-isolation` |
| Members HTTP inactive-module guard | Completed (local) | Members HTTP handlers and limiters omitted in profiles where Members is inactive; service required for Commerce session resolution is preserved | `R24-members-module-zero-construction`, `R17-wsl-preset-module-isolation` |
| Configurable workflow contract | Partially completed (local) | Bounded custom state and transition validation in Admin Studio; persistence and moderation UI remain module-owned | `R5-configurable-workflow` |
| Bounded Admin bulk contract | Partially completed (local) | 500-record bound, deterministic ID normalization, and fail-closed request envelopes; generated `bulk.ts` emits matching envelope; mutation and policy remain module-owned | `R5-bounded-admin-bulk-contract`, `R5-generated-bulk-request-adapter` |
| Opt-in Admin bulk list surface | Partially completed (local) | Accessible selection checkboxes and escaped action metadata in collections declaring `BulkActions`; mutations, CSRF, and policy remain module-owned | `R5-opt-in-admin-bulk-list-surface` |
| Bounded async relation contract | Partially completed (local) | 50 default / 100 hard item limit, bounded query/cursor, and deterministic ID/label projections; provider lookup and authorization remain module-owned | `R5-bounded-async-relation-contract` |
| Locale fallback chain | Partially completed (local) | Exact -> primary language -> supported default precedence with 128-locale bound and fail-closed normalization; integrated into public Pages and News services; module acceptance and edition evidence open | `R5-locale-fallback-chain`, `R6-pages-locale-fallback-integration`, `R6-news-locale-fallback-integration` |
| Commerce locale money rendering | Completed (local) | Product, variant, catalog, cart, and checkout surfaces utilize exact integer-cent locale formatting; float-style template formatting removed | `R28-commerce-locale-money-rendering` |
| Frontend selected design-system metadata | Completed (local) | Generated `frontend/starter.json` exports selected provider and supported provider-agnostic adapter catalog in distinct fields | `R29-frontend-selected-design-system` |
| DX iteration baseline | Partially completed (local) | Scaffolding, resource generation, and compatibility acceptance median of 1.191 s across 3 clean Windows runs; Payload comparison and second independent customer scenario open | `R30-dx-iteration-benchmark`, `R31-dx-iteration-benchmark-3-runs` |
| Headless starter error/idempotency parity | Completed (local) | Independent TypeScript starter checkout key validation, typed API errors/request IDs, and header + streaming bounded response protections aligned with generated client | `R32-headless-starter-error-idempotency` |
| WSL production-gate preflight (current commit) | Completed (local service) | Full/shuffled/race suites, reproducible build, and portable benchmark passed on PostgreSQL 18.6 + Redis/Valkey; real restore/proxy/key/capacity gates open | `R33-wsl-production-gates-current` |
| Headless starter client contract smoke | Completed (local) | Node smoke test verifies idempotency, typed error/request ID, standard response, and chunked 2 MiB bound for compiled TypeScript client | `R34-headless-starter-client-contract-smoke` |
| Headless starter E2E wiring | Completed (local/CI script) | `scripts/run-e2e.ps1` enforces `examples/headless-starter` contract smoke test prior to generated browser journeys | `R35-headless-starter-e2e-wiring` |
| Headless starter client bounds | Completed (local) | Contract smoke verifies 1–120000 ms fail-closed timeout boundaries | `R36-headless-starter-client-bounds` |
| Headless starter timeout abort | Completed (local) | Contract smoke verifies real `AbortController` timeout signal for pending requests | `R37-headless-starter-timeout` |
| Headless starter local verification gate | Completed (local/CI) | `scripts/verify-local.ps1` wires headless starter smoke tests into pre-push verification when npm is present; skips cleanly with explicit warning in Node-free environments | `R38-local-verification-headless-starter` |
| Local Compose loopback bindings | Completed (local) | PostgreSQL, Redis, and MinIO development ports bound strictly to loopback; services never exposed to LAN accidentally | `R39-local-compose-loopback-bindings` |
| Compose binding security lint | Completed (local/CI) | Loopback-only Compose bindings enforced fail closed via lint in `verify-local.ps1` | `R40-compose-security-lint` |
| Headless starter external abort | Completed (local) | Contract smoke verifies host `AbortSignal` cancellation propagates to internal request controller | `R41-headless-starter-external-abort` |
| Headless starter origin validation | Completed (local) | Client validates absolute HTTP(S) origin, credential, and scheme boundaries fail closed prior to request execution | `R42-headless-starter-origin-validation` |
| Headless starter per-request abort | Completed (local) | Public client methods support selective cancellation via per-call `HydroGoRequestOptions.signal` | `R43-headless-starter-per-request-abort` |
| Generated client abort/origin parity | Completed (local) | Project generator `HydroClient` output matches starter contract with HTTP(S) baseUrl validation and per-call read abort options | `R44-generated-client-abort-origin-parity` |
| Local Compose image pinning | Completed (local) | MinIO/Redis fixture images pinned to release tags; `:latest` usage rejected by Compose lint | `R45-compose-image-pinning` |
| Generated client stream cleanup | Completed (local) | Generated bounded response reader uses `finally` with `releaseLock()` to prevent stream lock leaks | `R46-generated-client-stream-lock` |
| Development Compose exact image pins | Completed (local) | PostgreSQL/Redis development fixtures pinned to exact patch/release tags; mutable major tags rejected by lint | `R47-compose-development-image-pinning` |
| Generated client request ID parity | Completed (local) | Generated `HydroAPIError.requestId` carries body/header request ID; oversized error paths preserve correlation | `R48-generated-client-request-id` |
| Generated client timeout parity | Completed (local) | Generated `HydroClient` verifies timeout values under identical 1–120000 ms fail-closed contract as starter | `R49-generated-client-timeout-parity` |
| Generated client relative-path boundary | Completed (local) | Generated `HydroClient` rejects absolute or protocol-relative mutation paths; prevents transmitting configured API keys to wrong hosts | `R50-generated-client-relative-path` |
| WSL PostgreSQL + Redis gate preflight | Completed (local service) | Verified full/shuffled/race suites, reproducible build, portable benchmark, and atomic limiter/TTL evidence on PostgreSQL 18.6 and Redis/Valkey | `R51-wsl-postgresql-redis-gates` |
| Local production drill refresh | Completed (local) | Isolated SQLite readiness, backup/release/portable contracts, and 20-repetition blue-green cache rollback stability passed | `R52-local-production-drills` |
| Generated client origin-bypass hardening | Completed (local) | Request paths attempting origin escape via backslash normalization rejected through resolved-origin comparison | `R53-generated-client-origin-bypass` |
| S3 object-key/public-base boundary hardening | Completed (local) | All blob operations enforce unified bounded key validation; public base cannot carry credentials, query, or fragments | `R54-s3-object-key-boundary` |
| S3 endpoint URL boundary hardening | Completed (local) | URL-formatted endpoints require HTTP(S) without credentials, query parameters, or fragments | `R55-s3-endpoint-boundary` |
| S3/CDN runtime configuration wiring | Completed (local) | `cdn_base_url` and `s3_force_path_style` config values wired into running S3 blob store; credential/query/fragment boundaries enforced | `R56-s3-config-wiring` |
| Public media CDN resolution | Completed (local) | Public Pages, News, and Commerce server-rendered images utilize optional provider URL; secure same-origin fallback preserved when absent | `R58-public-media-cdn-consumers` |
| S3/CDN public URL boundary | Partially completed (local) | S3-compatible path-style option, explicit HTTPS public base, and bounded object URL generation; 1s–24h presigned URL capability and protected `/api/media/{id}/url` endpoint added for private media; real provider/CDN and invalidation evidence remains open | `R6-s3-cdn-public-url-boundary`, `R6-bounded-private-media-urls`, `R6-protected-media-url-endpoint` |
| WSL local verification | Completed (local) | `go test ./... -p 1 -timeout=5m` on Ubuntu/WSL2 | `wsl-full-suite-2026-09-16-r5` |
| WSL static quality verification | Completed (local) | `go vet ./...` on Ubuntu/WSL2 | `wsl-vet-and-full-suite-2026-09-16` |
| WSL race detection | Completed (local) | `go test -race` on broker/runtime/release target packages | `wsl-race-broker-runtime-2026-09-16` |
| OCI container image definition | Completed (static local) | Non-root, credential-free amd64 Dockerfile, bounded build context, and daemonless lint gate | `R6-container-image-definition-2026-09-16` |
| Live Redis rate-limit atomic/TTL evidence | Completed (local service) | Initial request, limit rejection, and TTL verification via `HYDRO_TEST_REDIS_URL`; tests skip cleanly when service absent | `R6-redis-rate-limit-2026-09-16` |
| PostgreSQL production-gate automation | Completed (WSL service) | Full/shuffled/race suite, reproducible build, and portable benchmark against PostgreSQL 18.6 | `R6-postgres-production-gates-2026-09-16` |

## Components Not Yet Operating as Production Services

The SDK layer in this repository provides secure contracts and local artifact operations.
The following external dependencies cannot be deemed complete without real endpoints, accounts, and release infrastructure:

1. OAuth/device-flow authentication and token refresh service.
2. Artifact broker backed by a GitHub App or private registry.
3. Cross-platform signed release pipeline with public key rotation.
4. Official MSI, Homebrew, deb/rpm, and macOS notarization releases.
5. Offline licensing, revocation, mirror, and rollback rehearsals.
6. S3/CDN provider integration, backup/restore, and disaster recovery evidence.

The broker endpoint, grant, and error contracts are actionable and specified in [HYDRO_SDK_BROKER_API.md](HYDRO_SDK_BROKER_API.md); they are not an active backend service.

Placing user PATs into private GitHub repositories is not an acceptable substitute. Clients must receive only short-lived grants/URLs and download signed artifacts, never raw source code. See [`HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md`](HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md) and ADR-0091 for details.

## Progress by Phase

- **Phase 0 (benchmark):** HydroGo measured across two independent resource fixtures (median 1.651 s / 1.546 s, local Windows); Payload 3.90.1 measured on SQLite harness for build, cold/warm admin, and warm API smoke. Authenticated CRUD, p95/p99 load tests, and capacity parity remain open (`R102`).
- **Phase 1 (resource contract/CLI):** Partial. Project namespace, lock loading, and check workflows exist; generated collections self-describe via deterministic `spec.json`, and `hydro resource check --root` discovers and validates all scaffolds; global resources, field validation, lifecycle scaffolding, and bounded collection list transports were added; full rich query AST field/operator binding to generated transports and runtime wiring for migration/admin/API remain open.
- **Phases 2–3 (Admin/editorial):** Open. Product acceptance tests required for async relations, bulk actions, preview bridge, scheduling, and locale fallback.
- **Tenant-scoped resource generation:** Completed (at contract level). Blueprint `tenant_scoped` flag generates model/repository/service/transport surfaces with explicit tenant parameters and fail-closed resolvers; real PostgreSQL RLS, provisioning, quotas, and restore acceptance remain production gates (ADR-0235).
- **Admin bulk execution primitive:** Completed (at Core contract level). Added ordered, bounded execution with module-owned callbacks that do not leak error strings; authorization, idempotency, audit, and transaction semantics remain the responsibility of the Edition/module (ADR-0236).
- **Edition-extensible tenant RLS verification:** Completed (at contract level). Provisioning verifier validates generated tenant tables fail-closed; policy/migration emission and live PostgreSQL restore/capacity evidence remain operational gates (ADR-0237).
- **Phase 4 (kit ecosystem):** Partial. Kit and edition contracts exist; registry, dependency resolution, and compatibility matrices are incomplete.
- **Phase 5 (frontend SDK/starter):** Partial. Adapter contracts, typed client/query workflows, disposable SQLite demo, and local Chromium E2E evidence (47 tests) exist; external Client Edition and production authentication acceptance remain incomplete.
- **Headless client hardening:** Completed (local). Starter and generated clients enforce bounded timeouts and error body sizes, returning typed `HydroAPIError` with request IDs; live browser/auth acceptance remains a distinct gate.
- **Phase 6 (production/upgrade):** Open. Local production drills passed (SQLite readiness, backup/release/portable, and blue-green cache rollback); PostgreSQL capacity, real proxy/restore drills, and signed cross-OS release evidence remain pending.

## Local Verification Workflow

Generated Edition runtime checklist now documents the optional `ResourceAccess`
callback alongside the API authorization boundary. Edition authors are told to
derive both from the same user, tenant, and record rules; leaving the callback
nil retains legacy all-actions UI, while API authorization remains mandatory.

Site Admin appearance save has a registered HTTP/CSRF/SQLite persistence test:
invalid CSRF and explicit Admin permission denial leave branding unchanged, and
an authorized save round-trips all fields through the SQL store. Site module
tests and vet passed. This is one mutation path; the wider Admin role × route ×
persisted-state matrix and PostgreSQL route-level check remain open.

SEO settings now have the same persisted route boundary: unauthorized and
invalid-CSRF writes leave no setting, valid local paths are deduplicated and
stored, and invalid paths do not replace a prior valid value. Site tests and
vet pass; redirect-rule and other Site mutations remain unverified at this
route-level SQLite boundary.

The legacy redirect handler also has registered-route SQLite coverage: invalid
CSRF and explicit Admin permission denial persist nothing, while an authorized
Admin persists the redirect. Redirect-rule CRUD/import and other Site mutation
routes remain open.

Locale navigation is covered through the registered mutation route and SQLite:
an explicitly permitted Editor can persist a parent/child tree; Admin readback
includes hidden entries and the public projection excludes them. Invalid CSRF
and explicit Admin denial leave the table empty. Other Site mutations and
PostgreSQL route-level behavior remain open.

Social profile and contact-widget mutations now have matching CSRF/SQLite
route tests. Authorized writes persist and round-trip to the public projection;
invalid CSRF, Admin permission denial, an untrusted social host, and an invalid
phone do not create or overwrite data. The rest of the Site mutation matrix and
PostgreSQL route-level behavior remain open.

Maintenance mode also has route-level SQLite coverage: unauthorized writes do
not persist, scheduled maintenance and allowlist fields round-trip, the private
preview key is stored as a hash, and malformed IP ranges cannot overwrite a
valid setting. Site tests and vet pass; other Site mutations and PostgreSQL
route coverage remain open.

The custom 404 and 500 editors now have route-to-public-response SQLite journey
coverage: the 404 appears through the public catch-all, the 500 renderer keeps
its 500 status, denied writes have no effect, and an external link is rejected
without replacing the saved 404. Remaining Site routes are still open.

Internal staff broadcast settings are covered through the Admin route and
SQLite. The notice persists privately, never appears in the public settings
projection, and invalid severity or unauthorized/CSRF-denied writes leave the
previous value unchanged. Remaining Site mutations and PostgreSQL route-level
tests remain open.

ADR-0324 custom-code admin routes now have SQLite/CSRF integration coverage:
only an active Admin can save; an Editor with `site.manage=true` is denied;
approved code round-trips to the editor but remains private; unapproved hosts
cannot replace the saved value. Other Site route state and PostgreSQL route
coverage remain open.

The translation editor now has a tenant-scoped route test: denied CSRF/role/
permission combinations write nothing, a permitted save is visible through the
tenant's i18n catalog projection, regional locale tags canonicalize, and
unknown keys cannot overwrite persisted translations. PostgreSQL and complete
browser translation journeys remain open.

Redirect rule create/read/delete now has a registered HTTP/CSRF/SQLite
workflow test. Admin can persist a wildcard rule and remove it; denied CSRF,
Editor delete, and explicit Admin permission denial leave state unchanged.
CSV import and PostgreSQL route-level evidence remain open.

CSV redirect import now has multipart route coverage: role/CSRF denials write
nothing, valid batches persist fully, and a later invalid row prevents the
entire batch from changing prior data. Live registry/cache and PostgreSQL route
tests remain open.

Form Builder schema create/update/delete now exercises the real HTTP routes and
SQLite form registry. Role/CSRF denials are side-effect free, invalid schemas
preserve the current version, and authorized CRUD round-trips. Public form
submission and inbox/lead persistence remain open.

Public form submit → idempotent inbox → Admin triage is now one route/SQLite
journey: replay creates no duplicate, key reuse with changed input conflicts,
inbox HTML omits internal hashes, Editor/CSRF/permission denials cannot mutate,
and authorized status/lead assignments persist. Exports, browser UX and
PostgreSQL route-level behavior remain open.

Lead-note create/list is also covered: only permitted Site managers can add a
note, denied roles and CSRF persist nothing, and returned records retain the
author identity. Inbox CSV/XLSX downloads now have registered-route SQLite
coverage: Admin downloads persisted submissions; Editor is denied; CSV guards
formula-like values and XLSX stores them as inline text. Browser workflows and
PostgreSQL route behavior remain open.

The Form Builder working-hours route now has SQLite authorization, CSRF,
validation and persisted-state coverage: denied requests do not write, invalid
timezone or reversed interval does not replace settings, and an authorized
Admin's Istanbul schedule reads back exactly. Browser interaction and
PostgreSQL route behavior remain open.

Nested navigation's tenant-scoped round-trip now also goes through its
registered route and production CSRF middleware. CSRF, Editor and explicit
permission denials preserve the original menu; an authorized Admin persists
the nested relation and target. Browser drag-and-drop and PostgreSQL route
behavior remain open.

The 404 report conversion now likewise runs through the registered route and
CSRF middleware: denied and invalid requests preserve the report and do not
register redirects; authorized conversion persists the redirect, updates the
runtime registry and clears the report. Browser interaction and PostgreSQL
route behavior remain open.

On Windows:

```powershell
go test ./internal/sdkdist
go test ./internal/projectgen ./internal/release ./cmd/hydro
```

On WSL Ubuntu (`whoami` must be `operator`):

```bash
cd /path/to/project
go test ./... -p 1 -timeout=5m
```

Test results and observation duration must be added as a dated report under
`docs/validation/`. Passwords or tokens must never be written to reports.

2026-10-09 Admin Studio E2E recheck: `e2e` dependencies were installed
offline from npm cache and Playwright discovered the two independent Edition
CRUD/media journeys. The SQLite fixture opened/migrated and composed its
routes, then could not bind `127.0.0.1:8099` (`socket: operation not
permitted`). Browser assertions remain unexecuted; run on a loopback-capable
runner. `HYDRO_TEST_POSTGRES_URL` is unset in this checkout, so no PostgreSQL
result is inferred.

Site font localization's registered HTTP route now has a deterministic SQLite
journey with a test-only provider transport. Role/CSRF/permission denials and
unsafe family validation happen before provider calls or file/settings writes;
the authorized flow stores the allowlisted font files and public CSS setting.
The real Google Fonts provider is not contacted by the test. A temporary
PostgreSQL cluster initialized but could not bind a Unix socket in this runner;
PostgreSQL behavior remains unverified here.

Full root regression recheck then passed: normal and shuffled `go test ./...`,
`go vet ./...`, moduleguard, migrationguard (110 migrations), and all three
Edition importguards. This run still had no `HYDRO_TEST_POSTGRES_URL`; see
[`admin-studio-root-suite-2026-10-09.md`](validation/admin-studio-root-suite-2026-10-09.md)
for exact commands and the PostgreSQL/browser limits.

Because the root module does not include nested modules, `tests/external-edition`
was independently rechecked: all its tests, `go vet ./...`, and its Edition
importguard passed. This is SQLite/in-process composition evidence only; its
live browser journeys and PostgreSQL adapter remain open.

Site redirect runtime/persistence consistency: redirect cycles are validated
before saving, then the durable write and in-memory snapshot publication are
serialized. Legacy saves, redirect-rule CRUD/import and 404 conversion share
this path; route tests prove rejected cycles and failed writes do not leave a
persisted/runtime mismatch. Root/Site tests, vet and focused race tests passed.
Multi-instance invalidation (ADR-0268), PostgreSQL and browser gates remain
open; see `docs/validation/site-redirect-runtime-consistency-2026-10-09.md`.

Site Admin route-derived authorization coverage now includes every registered
non-public GET deep link with Editor and Viewer principals; both are rejected
before route data access. Existing route-derived POST checks cover invalid
CSRF and denied/inactive principals. Focused and full Site transport tests and
package vet passed. Other Admin modules, browser menu parity, PostgreSQL, and
customer deployment behavior remain open; see
`docs/validation/site-admin-deeplink-role-matrix-2026-10-09.md`.

The route audit also fixed an Admin permission bypass in Site custom-code
management: an explicit `admin.site.manage=false` override now denies the
otherwise privileged Admin role. SQLite route tests prove no private setting
is written on denial and authorized save/read still works. Full Site module
tests and transport vet passed; PostgreSQL/browser proof remains open. See
`docs/validation/site-custom-code-permission-override-2026-10-09.md`.

Dynamic Collections role coverage now dispatches every registered HTTP method
and concrete route path through the actual protected router, rather than
calling handlers at `/`. Active Editor and Viewer principals receive 403 for
all routes; the existing route-derived CSRF matrix covers every mutation.
Full module tests and transport vet passed. This is in-process evidence, not
browser or deployed Edition validation; see
`docs/validation/dynamic-collections-route-role-dispatch-2026-10-09.md`.

Media cleanup now honors an explicit `admin.media.manage=false` override as
well as its Admin-role requirement. Route authorization tests cover all media
Admin GET/POST handlers; a SQLite-backed cleanup test proves denied deletion
does not remove assets, while the authorized Admin flow remains functional.
Full media tests and vet passed. Browser/PostgreSQL remain open; see
`docs/validation/media-cleanup-permission-override-2026-10-09.md`.

Core Admin identity and permission packages were re-run after the recent
management-route changes: `go test ./internal/auth/... -count=1 -p 1` and
`go vet ./internal/auth/...` passed. This package-local evidence does not
certify PostgreSQL (`HYDRO_TEST_POSTGRES_URL` is unset) or every Admin route;
see `docs/validation/admin-auth-route-regression-2026-10-09.md`.

The five Admin account mutation routes now have an additional route-dispatch
matrix for Editor, Viewer, inactive Admin, explicit `users.manage` denial, and
invalid CSRF. The focused test and complete auth/sqlstore suite pass; browser
and PostgreSQL remain open.

Token lifetime settings now have a matching `Protected` route plus CSRF
middleware test: invalid CSRF and Editor requests return 403 without writing,
while a valid Admin request persists the setting. This closes the previous
direct-handler-only CSRF evidence gap for that route; full server/browser
coverage remains open. See
`docs/validation/admin-token-lifetime-route-security-2026-10-09.md`.

Extension channel bind/revoke now have route-level tests for valid CSRF, Admin
role, active session ownership, tenant context, and no-write denial. Client
JSON cannot set the authority principal, and registry errors are not exposed.
Full `cmd/hydro` tests and vet passed. Browser/deployed-session coverage remains
open; see `docs/validation/admin-extension-channel-route-security-2026-10-09.md`.

Generated Editor.js forms now have a planned eight-variant theme-family/light-
dark axe browser matrix. Playwright discovered the suite, but its fixture
server failed before tests ran because loopback bind returned `EPERM`; no
accessibility result is claimed. The spec now supports a separately started
loopback fixture for restricted browser runners; the CI browser job also runs
the full Playwright suite on Windows. See
`docs/validation/generated-block-editor-theme-matrix-2026-10-09.md`.

The Blueprint generator and both the native Project Edition and separate
external Edition were revalidated: generator, Edition and external module tests
plus vet passed; `hydro project check`, moduleguard, migrationguard and external
importguard passed. This run covered SQLite/local contracts only. PostgreSQL
was not available (`HYDRO_TEST_POSTGRES_URL` unset; Docker API denied), and the
authenticated Chromium gate remains open. Evidence:
`docs/validation/blueprint-runtime-regression-2026-10-09.md`.

Generated CRUD API failures now render safe Turkish/English localized copy
based on the Admin locale. A failure injection with a database URL, host,
filesystem path, and panic text verifies none reach the response and no
success redirect is issued. Full Admin Studio/i18n tests and vet passed. The
plain-body error UX, browser verification, and PostgreSQL remain open; see
`docs/validation/generated-admin-error-localization-2026-10-09.md`.

Create/update transport or API 5xx failures keep the user in the shared Admin
shell, show an accessible localized alert, and preserve submitted values and
CSRF. Delete failures now restore the record's edit form and delete action when
the record can still be read; otherwise they show delete-specific localized
guidance. Turkish/English tests cover all three operations and the unavailable
record fallback. Browser/PostgreSQL remain open; see
`docs/validation/generated-admin-save-recovery-2026-10-09.md`.

## Next Safe Steps

1. Complete Payload benchmark with authenticated browser CRUD, seed data, and p95/p99 load testing.
2. Expand resource contract migration/admin/API runtime wiring coverage based on benchmark measurements.
3. Accept real service decisions for login/broker via ADRs.
4. Verify signed release pipeline and installers against clean environments.
5. Execute production capacity, restore, and upgrade drills.


2026-10-09: The full root Go suite also passed with `HYDRO_TEST_POSTGRES_URL`
pointing at a disposable PostgreSQL 18.6 cluster; backup fixtures used the
temporary test login as their configured privileged role. This is local
root-module PostgreSQL evidence, not nested Edition or production deployment
certification. See `docs/validation/admin-studio-full-go-suite-2026-10-09.md`.

2026-10-09: Generated Admin article/story list and forms, plus story draft
inbox, passed 40/40 four-theme × light/dark axe checks on each adapter. Full generated-richtext
Chromium suite passed 9/9 on SQLite and PostgreSQL 18.6 fixture storage. This
adds generated editor visual/accessibility evidence; it does not cover every
product Admin route or certify deployed Edition migrations. See
`docs/validation/generated-admin-theme-form-matrix-2026-10-09.md`.

2026-10-09: Admin-management E2E dynamically inventories rendered privileged
POST actions from ten management pages. Editor and Viewer valid-CSRF requests
were denied on every collected action. The complete Admin-management browser
file passed 5/5 on SQLite and PostgreSQL 18.6. This does not cover unrendered
API-only actions, every module POST, or every Admin permission override.
Evidence: `docs/validation/admin-permission-matrix-2026-10-08.md`.

2026-10-09: All six registered member-management POST routes now test Editor,
Viewer, inactive Admin, explicit `admin.members.manage` deny, and missing or
mismatched CSRF. Denials leave member fields and active sessions unchanged.
Member transport package tests and vet passed; no database-adapter claim is
made for this in-memory handler matrix. Evidence:
`docs/validation/admin-permission-matrix-2026-10-08.md`.

2026-10-09: Member list, CSV export, XLSX export and detail GET routes now
exercise 24 `admin.members.manage` role/override cells. The member transport
package and vet passed. This matrix uses in-memory handler contracts and does
not claim database-adapter coverage. Evidence:
`docs/validation/admin-permission-matrix-2026-10-08.md`.

2026-10-09: Generated locale-copy authorization tests count callback calls so
Editor/Viewer and missing-CSRF denials prove no extra persistence call. Focused
test, full `internal/adminstudio` package, and vet passed. This is in-memory
callback evidence, not database-adapter certification. See
`docs/validation/admin-permission-matrix-2026-10-08.md`.
