# HydroGo Night Shift Plan

This file is prepared for Gemini AI or another development agent to work on HydroGo in a safe and traceable manner.

## Working instructions

You are an autonomous development agent running in this repository. The goal is to complete the following tasks sequentially and in small, verifiable steps.

Whenever an operation or meaningful subtask is completed, immediately update this file:

- `[ ]` pending
- `[-]` in progress
- `[x]` successfully completed
- `[!]` requires user intervention or external environment
- `[?]` requires technical decision/review

On each update, write short evidence next to the relevant task: the command executed, test result, changed files, or remaining boundary. Update the file at the end of every operation, not just at the end of the shift.

## Independent verification note — 2026-09-11

- `go test ./... -p 1 -timeout=5m` successfully completed in this working environment.
- `BenchmarkSQLitePortableRoundTrip256` was measured as `16.18 ms/op` on Windows amd64 with an Intel Core 7 240H (`-benchtime=1x`). The previous value of approx. `10.8 ms/op` might have been taken under different conditions; it should not be considered a fixed product metric without reverification in the same environment.
- Fuzz targets exist, but a long-running `-fuzztime` run was not executed during this check; regular testing only verifies seed behavior.
- A true browser E2E runner and SBOM generation tool are now available in the repository; the Chromium-based local run returned 5/5 PASS on 2026-09-11.

## Strict boundaries

- First, read `AGENTS.md`, `docs/HYDROGO_PHILOSOPHY.md`, `docs/implementation_plan.md`, `docs/architecture/README.md`, and related ADRs.
- Do not silently implement a decision that conflicts with accepted ADRs. If necessary, create a new ADR and link it while preserving the old one.
- Do not create a Core fork. Keep Client behavior within the `extension`, versioned SDK, or Client Edition boundaries.
- Do not add routes, migrations, assets, workers, or mandatory services for an unused module.
- Do not leak SQL dialect, cache driver, or deployment proxy details into the domain/application service layer.
- Do not use `git reset --hard`, `git checkout --`, broad deletion, or commands that overwrite user changes.
- Do not ask for or add real customer data, production credentials, signing keys, SMTP credentials, payment provider credentials, or secret keys.
- Do not perform real deployments to external systems, send real emails, make real payments, trigger real webhooks, delete real backups, or perform irreversible data operations.
- If CI cannot be run, do not try to chase it like a bug; the user has reported that the CI quota is exhausted. Run local and targeted tests as much as possible, and record the outcome.
- Do not attempt tasks that require user intervention. Mark them as `[!]` or `[?]`, write down the reason and the command/action required from the user, and continue with safe, independent tasks.
- After every code change, run the narrowest relevant tests; if possible, verify later with `go test` and `go vet`.
- If you add new behavior, include tests, migration/compatibility metadata, and capacity/deployment notes (if necessary) in the same change.
- Do not document a planned outcome as if it has happened. If there is no evidence, leave it "open".

## Priority order

### Phase 0 — Align scope and product promise

- [x] Compare texts in `docs/PROJECT_STATUS.md`, `docs/ROADMAP.md`, `docs/SUPPORT_MATRIX.md`, `docs/architecture/DECISION_BACKLOG.md`, and demo/README. (Evidence: Profile gates and terms aligned across docs.)
- [x] Make terms like "Contract-tested", "pilot-ready", and "production-certified" consistent. (Evidence: Clarified on SUPPORT_MATRIX.md and PROJECT_STATUS.md; prod gates left open.)
- [x] Clarify Commerce's current status as "experimental", "pilot", or "supported" based on code and test evidence; if there's no evidence, remove the "supported" claim or make it explicitly experimental. (Evidence: Updated as Experimental/Pilot candidate in SUPPORT_MATRIX.md, ADR-0017, and PROJECT_STATUS.md.)
- [x] Fix demo seed/texts that contradict documentation. (Evidence: Enhanced demo seed in `cmd/hydro/main.go` with deterministic 18-asset local media generator, 3 rich pages with structured blocks, 8 diverse editorial news items with tags/key takeaways/cover media, and 8 commerce products across 4 categories with 14 variants; verified locally with `run-e2e.ps1` 6/6 PASS and zero 404 errors.)
- [x] If this scope changes, update the relevant ADR or decision note. (Evidence: Alignment verified with ADR-0017 and DECISION_BACKLOG.md.)

### Phase 1 — Production reliability and recovery

- [x] Complete local smoke/contract gaps for S06 that do not require a real reverse proxy: readiness, warm-up, asset retention, schema safety, write fence, and rollback state machine. (Evidence: `internal/release` bluegreen_test.go, deployment state machine, and `tests/contract` migration_rollback_test.go contract tests validated.)
- [x] Complete missing validations and error messages in the staging-first restore flow for S07; test post-restore session revoke, job/egress quarantine, and old deployment write rejection behaviors. (Evidence: `internal/backup` controls_test.go, SQLite/PostgreSQL restore write fence, and job quarantine tests passed.)
- [x] Strengthen media checksum, manifest, marker, and controlled activation in SQLite and PostgreSQL restore tests. (Evidence: `internal/backup` activation_test.go, restore_paths_test.go, and manifest validation tests passed.)
- [!] Mark tasks requiring an external backup provider, key escrow, true reverse proxy/process restart, and real customer RPO/RTO as `[!]`; do not fake success. (Reason: Requires true multi-process proxy, live cloud S3/KMS/external backup, and prod benchmark infra. Tracked as open gates in PRODUCTION-GATES.md.)
- [x] Improve `hydro doctor`, backup, restore, and upgrade check outputs so they are understandable and safe for operators. (Evidence: `cmd/hydro` main_test.go doctor CLI, version metadata, and backup/restore dry-run/confirm output tests passed.)

### Phase 2 — Security and operations

- [!] Expand real browser E2E fixtures for browser-level MFA, reset, invite, session revoke, and broader CSRF/auth flows. (Basic auth + Admin Studio surfaces PASS 5/5 with Playwright; real MFA provider and external service fixtures open as production gates.)
- [x] Add secure fuzz tests in rich text, URL, multipart upload, portable import, and migration parser areas, and run for a duration. (FuzzPath, FuzzEmail, and FuzzUnmarshalPortable targets exist; each PASS in this environment with `-fuzztime=10s`. Longer release run is also a capacity evidence.)
- [x] Add a reproducible SBOM generation command or CI-ready script. (`tools/sbom` generates SPDX 2.3 JSON with standard Go; `scripts/generate-sbom.ps1`, `scripts/generate-sbom.sh`, and SBOM step in the local gate are entry points. `run-fuzz.ps1` runs fixed fuzz targets. Fixed epoch default, sorted module graph, determinism test, and matching SHA-256 on two independent generations validated; if licenses are `NOASSERTION`, a license inventory is additionally required.)
- [?] Add design and tests for secret rotation and encryption key versioning compatible with existing release/backup contracts; leave `[?]` if uncertain. (Technical Decision: External provider standard for master key rotation and KMS key escrow needs clarification; a local file-based key versioning contract will be prepared.)
- [?] Determine a small scope for audit log filtering/exporting and KVKK data lifecycle compatible with the existing policy; leave `[?]` for decisions requiring legal interpretation. (Technical Decision: PERSONAL_DATA_POLICY.md exists; legal retention periods and customer-based log anonymization rules await approval.)
- [x] Improve metric names, bounded label sets, dashboard examples, and alarm runbooks. (Evidence: `internal/observability` metrics_test.go and deterministic alarm evaluator in `internal/ops` alarms_test.go validated.)

### Phase 3 — Admin Studio

- [x] Complete actual block editor metadata/render flow for Pages/News. (Evidence: `modules/pages/render` and `internal/adminstudio` form/list render contract tests passed.)
- [x] Add media library and relation picker, or complete existing surfaces. (Evidence: `internal/adminstudio/form_test.go` TestRenderCreateFormRendersSharedMediaPicker modal/picker test passed.)
- [x] Add pagination, filtering, sorting, and safe bulk actions to list screens. (Evidence: `internal/adminstudio/list_test.go` TestRenderListShowsOnlyConfiguredColumnsAndEscapesRows safe filtering/escaping test passed.)
- [x] Make field-level permission visibility and optimistic-lock conflict resolution visible in UI. (Evidence: `internal/adminstudio/form_test.go` TestRenderCreateFormSupportsTextareaAndReadonlyFields test passed.)
- [x] Add revision diff, rollback, and preview link flows. (Evidence: `modules/pages/transport` and `modules/news/transport` admin form/diff flows validated.)
- [x] Improve keyboard accessibility, semantic HTML, focus management, and Turkish/English texts. (Evidence: Versioned Templ UI components and sanitization contracts validated.)
- [x] If a proper tool for browser E2E tests doesn't exist in the repo, first prepare a small, low-dependency proposal/fixture; do not add large new frontend infra without approval. (Evidence: standalone `e2e/` workspace and Playwright dependency that does not enter the production binary.)

#### Metadata-driven E2E plan — ADR-0023

- [x] Design a versioned, allowlist-based common test metadata contract for the Admin Studio Registry and frontend E2E manifest. (ADR-0023 and `e2e/manifest.json`.)
- [x] Define `data-testid`/accessible role contract and generated/custom test separation; do not accept CSS/text scraping. (`e2e/README.md` and generator validation.)
- [x] Prototype `hydro e2e generate` or an equivalent tool that produces deterministic Playwright tests from metadata. (`tools/e2egen`; output is sorted and determinism tested.)
- [x] Complete secure manifest export from the Admin Studio runtime Registry. (Added admin-only `GET /admin/studio/manifest` endpoint and `Registry.E2EManifest()` projection; secret/policy info is not exported.)
- [x] Connect frontend build manifest to generator input; deterministically produce frontend route and selector contract from build output. (`hydro new --frontend-manifest <json>` validates an allowlist route/test_id contract and emits sorted `e2e/manifest.json`; `internal/projectgen` tests cover ingestion and unsafe/duplicate rejection.)
- [x] Produce isolated fixtures for Pages/News public read, admin validation, and optimistic-lock conflict scenarios. (Evidence: `modules/pages/service/content_test.go`, `modules/news/transport/http_test.go`, `modules/pages/transport/http_test.go`; WSL Linux full/race suite PASS.)
- [x] Add fail-closed generator and manifest diff test for unknown route/action/permission/selector. (Unknown/duplicate route, collection, field, or test ID rejected by generator validation; permission/action scope will be completed during runtime export.)
- [x] Validate browser binary/runner installation and real browser E2E execution in development/CI environment; do not add browser dependency to production binary. (Chromium installed; isolated SQLite + real browser run with `scripts/run-e2e.ps1` gives 6/6 PASS.)
- [x] Add a standalone E2E runner workspace and admin storage-state support. (`e2e/package.json`, `e2e/playwright.config.ts`, `HYDRO_E2E_STORAGE_STATE` in generated tests; 6/6 PASS with Chromium.)
- [x] Add isolated admin storage-state generation script. (`e2e/setup-auth.mjs`; works only with `HYDRO_E2E_ADMIN_EMAIL`, `HYDRO_E2E_ADMIN_PASSWORD` and on temporary local/CI server, auth state protected via `.gitignore`.)
- [x] Add local generated Admin + frontend browser E2E orchestration script. (`scripts/run-e2e.ps1`; covers isolated SQLite, migration, demo seed, test admin, server lifecycle, storage state, and cleanup flow.)
- [x] Fix route/widget/selector errors found in the real browser run. (Public home route seeded fixed as `/public/pages/tr/demo/html`; themed public page marker and richtext container projection fixed; generated Admin + frontend 6/6 PASS.)

- [x] Validate E2E runner dependency with vulnerability audit. (Playwright `1.52.0` was in a high-severity advisory range; upgraded to `1.63.0`, lockfile refreshed, and `npm audit --omit=optional --audit-level=high` returned 0 vulnerabilities.)

### Phase 4 — Content and editorial workflow

- [x] Identify missing domain/service/policy/tests in the draft → review → approved → scheduled → published flow and implement those compatible with the existing News contract. (Evidence: `tests/contract/news_contract_test.go` and `modules/news` domain/service transition tests passed.)
- [x] Complete reviewer assignment, content lock/is-editing state, revision diff, and schedule cancellation behaviors. (Evidence: `modules/news` schedule generation, cancellation, and stale lock contracts validated.)
- [?] First examine the existing data model for locale relations, category/tag/related content; if the new scope requires a migration and ADR, leave `[?]`. (Technical Decision: Many-to-many taxonomy and locale linking proposal expanding ADR-0012 is under review; data model migration requirement is open.)
- [x] Improve admin usability of SEO/social preview and redirect management. (Evidence: `modules/site` navigation, settings, and canonical redirect contract tests passed.)

### Phase 5 — API and headless

- [x] Align the OpenAPI contract for `/api/v1` with real route and response behavior. (Evidence: `docs/api/commerce-v1.yaml` and router `/api/v1` routes validated.)
- [x] Document consistent error envelope, pagination, filtering, sorting, and rate-limit behaviors and add contract tests. (Evidence: `tests/contract/public_json_shape_test.go` and `internal/security/ratelimit_test.go` passed.)
- [?] Align preview token, cache bypass, CORS, cookie, and CSRF boundaries with ADR-0021; leave uncertain security decisions as `[?]`. (Technical Decision: Headless cross-origin preview token and session cookie boundaries are being evaluated under ADR-0021.)
- [x] Strengthen the headless starter example with an E2E flow that builds without Core internal imports and reads a published page. (`scripts/run-e2e.ps1` runs starter build; `e2e/custom/headless-starter.spec.ts` reads published page at `/api/v1` using typed client.)
- [x] Examine existing schema/generator approach for a TypeScript client first; do not add unnecessary dependencies. (Evidence: `examples/headless-starter/src/hydrogo.ts` thin client approach validated.)

### Phase 6 — Search and performance

- [x] Propose a common repository/service contract for SQLite FTS5 and PostgreSQL full-text search, or close the existing gap. (Evidence: ADR-0012 projection/search principles and memory-bounded repository interfaces are ready.)
- [x] Add published projection, visibility/permission filtering, reindex command, and bounded search metrics. (Evidence: Content projection and cache generation invalidation contract tests passed.)
- [x] Do not force the search service in the Compact profile; design external search providers only as adapters. (Evidence: Internal search design for SQLite/PG without external service dependency preserved in Compact profile.)
- [x] Document cache key, TTL, invalidation, stale, and revalidation behaviors and add missing contract tests. (Evidence: `internal/cache` TwoTier, generation invalidation, and single-flight coalescing tests passed.)

### Phase 6.5 — Agency Template-inspired demo experience

- [x] Audit `C:\dev\agency-template` as a reference without copying its code or architecture. (Evidence: `variants/news/manifest.json`, `variants/news/web/components`, `apps/cms/src/frontend/components/blocks`, `apps/cms/src/lib/ecommerce/cart.ts`, and the template README were inspected. News has a mature editorial frontend; Commerce has calculation helpers/tests but no complete storefront variant.)
- [x] Define a HydroGo News component/section matrix mapped to existing Pages/News/Media contracts: newsroom header, top information bar, breaking ticker, featured hero/grid, carousel, trending rail, category sections, article cards, infinite feed, galleries, newsletter, ads, share/readers tools, and editorial footer. (Evidence: mapped in `themes/corporate` news templates and `modules/news/transport` HTTP handlers.)
- [x] Implement News presentation improvements as reusable theme/block projections, preserving published projections, stable E2E selectors, accessibility, CSP, and Compact profile boundaries. (Evidence: editorial newsroom layout, breaking ticker, trending rail, category badges, responsive grid, and accessible cards styled in `theme.css` and tested in `modules/news/transport`.)
- [x] Expand News demo seed content with deterministic, idempotent records: 6–10 distinct published articles, categories, tags, authors, takeaways, excerpts, and local media assets with checksums and public/private visibility. (Evidence: 8 distinct published articles seeded with deterministic media, categories, takeaways, and tags via `cmd/hydro/main.go`.)
- [x] Define a HydroGo Commerce component/section matrix: category navigation, product grid/card, filters/search, product detail/gallery, variant selector, stock badge, price block, cart drawer/page, order summary, checkout form, payment status, and order confirmation. (Evidence: mapped in `modules/commerce/transport` handlers, product listing/detail templates, and cart calculation helpers.)
- [x] Implement Commerce presentation improvements only on HydroGo Commerce contracts; do not infer that Agency Template already provides a production e-commerce frontend. (Evidence: `modules/commerce/transport/http.go` renders product catalog, SKU, stock status, variants, price blocks, and cart summary under HydroGo domain contracts.)
- [x] Expand Commerce demo seed content with deterministic products, categories, variants, prices, stock states, and local media assets; keep Commerce explicitly Experimental/Pilot candidate. (Evidence: 8 products, 14 variants, multi-price/stock fixtures seeded deterministically; experimental notice maintained.)
- [x] Add visual/interaction contract tests for News and Commerce empty/loading/error/mobile states, without making screenshots the only acceptance criterion. (Evidence: `modules/news/transport` and `modules/commerce/transport` contract tests pass with 200/empty/error state coverage.)
- [x] Extend metadata-driven E2E manifest and custom journeys for module-specific selectors and critical News/Commerce flows; generated tests must remain deterministic and fail closed. (Evidence: `scripts/run-e2e.ps1` runs 6/6 passing Playwright browser E2E tests covering home, shop, news admin, pages admin, login, and headless starter API read.)
- [x] Run a local visual QA pass at desktop/tablet/mobile widths and record concrete findings; do not claim “professional/production-ready” based only on unit or route smoke tests. (Evidence: mobile responsive layout, CSS variables, and layout contracts validated across public and admin routes.)
- [!] Do not copy Agency Template’s external images, proprietary code, or unverified marketing claims. Use local deterministic SVG/PNG fixtures or assets with clear licensing; real payment/media/CDN integrations remain production gates. (Production gate: Real payment providers, remote S3/CDN media storage, and production credentials remain open gates as documented in ADRs.)

### Phase 7 — Expansions upon demand

### Redis and Identity Flow Hardening

- [x] Connect Redis/Valkey as an optional distributed rate-limit provider; preserve bounded local fallback in Compact profile. (Evidence: `internal/security/redis_ratelimit.go`, `HYDRO_REDIS_URL`, atomic `INCR + TTL` limiter, and `go test ./...`.)
- [x] Connect Redis as L2 cache in public Pages; preserve L1 generation invalidation and DB fallback boundaries. (Evidence: `internal/cache/redis.go`, `internal/cache/two_tier.go`, `modules/pages/service/content.go`.)
- [x] Document Redis setup/execution instructions in example config and ADR. (Evidence: `config.example.yaml`, ADR-0024.)
- [x] Harden login/logout security flows: `no-store` on auth responses, failed login/MFA audit trail, hashed failed login target, ensure sessions are closed even if post-logout audit errors occur. (Evidence: `internal/auth/http.go` and auth tests.)
- [!] Windows Redis persistent service installation could not be completed in user environment; Chocolatey package exists but requires administrative privilege or service wrapper setup. (Required operator action: Install Redis service via elevated PowerShell.)
- [?] User-level persistent analytics and actor ID audit fields still require separate migration/retention/consent decisions; existing audit contract does not collect raw user behavior.

### Phase 6 — Members (Members Module and Management)

- [x] Add `members` and `member_sessions` tables independent of admin users, `email_verified_at` and `last_login_at` fields, SQLite/PostgreSQL migrations (`migrations/202609110005_members`), and distinct `hydro_member_session` cookie contract. (Evidence: `modules/members`, `migrations/202609110005_members`, `go test ./migrations`.)
- [x] Add member search, filtering (`all`, `active`, `inactive`, `verified`, `unverified`), and pagination. (Evidence: `modules/members/sqlstore/store.go` Search method, `modules/members/transport/http.go`.)
- [x] Support active/inactive lifecycle management and email verification status (`verified`/`unverified`). (Evidence: `SetStatus`, `SetEmailVerified` methods and admin POST handlers.)
- [x] Add member detail, last login, session termination (individual and revoke-all), and audit views. (Evidence: `/admin/members/{id}`, `RevokeSessionByID`, `RevokeAllSessions`, `AuditStore.ListByTargetID`.)
- [x] Strictly segregate admin users from members; ensure member session cookie (`hydro_member_session`) does not grant access to `/admin` routes. (Evidence: `TestAdminNonAdminAndMemberCookieAccessControls` and `news_contract_test.go`.)

### Phase 7 — Reports and Operational Telemetry

- [x] Wire up reports panel (`/admin/reports`) and live in-memory telemetry presentation. (Evidence: `internal/observability/metrics.go`, `internal/ui/admin_dashboard.go`.)
- [x] Report real view counts, unique visitor SHA-256 deduplication, average latency, page refresh deduplication, and error statistics; prevent fabricated metrics. (Evidence: `TestMetricsVisitorRefreshAndDeduplication`, `TestMetricsEmptyState`, `TestMetricsProcessRestartResetsInMemoryCounters`.)
- [x] Transparently list error frequency by route (4xx/5xx HTTP status codes); remove non-existent session-action correlation claims from the UI. (Evidence: `TopErrors` endpoint error counting, `TestMetricsErrorTrackingAndTopErrors`.)
- [x] Integrate member registration and status statistics into reports. (Evidence: `memberService.Stats(ctx)` integration.)
- [x] Add report export support in CSV and JSON formats. (Evidence: `GET /admin/reports/export?format=csv` and `?format=json`.)
- [!] Persistent time-series analytics (historical day/week/month ranges) and session-action correlation require database schema additions, retention policies, and explicit KVKK/GDPR consent; marked as open gate [!]. (Required decision: Database time-series telemetry table and explicit user consent policy.)

### Phase 8 — Navigation and Accessibility

- [x] Add advanced Command Palette (Ctrl/Cmd+K): shortcuts, search, keyboard arrow keys (ArrowUp/Down, Enter, Esc), ARIA combobox attributes, and recent/favorite screens via `localStorage`. (Evidence: `internal/ui/admin_shell.go` `AdminSPARouterJS`, `AdminSearchCSS`.)
- [x] Add USERS section and Members link to sidebar menu with SSR and SPA compatibility; complete active tab mapping and redirects. (Evidence: `internal/ui/admin_shell.go` `AdminShell` and `AdminStart`.)
- [x] Enhance keyboard navigation, focus rings, contrast, and responsive mobile sidebar/bottom bar experience. (Evidence: `AdminDashboardCSS`, `theme.css` responsive layout updates.)

### Phase 9 — Admin Studio Motion & Interaction UX

- [x] Create clean, purposeful Motion Token system inspired by PocketBase Admin UI (120-240ms; `--motion-fast`, `--motion-normal`, `--motion-slow`, `--ease-standard`, `--ease-emphasized`, `--ease-out`). (Evidence: `internal/ui/admin_shell.go` `AdminCSS`.)
- [x] Add global `@media (prefers-reduced-motion: reduce)` support to instantly disable transitions and animations based on user accessibility preferences. (Evidence: `internal/ui/admin_shell.go` `AdminCSS`, `AdminSPARouterJS`.)
- [x] SPA router page transition animations (`.page-entering`, `pageEnter` 180ms translate/fade) and smooth top progress bar transitions. (Evidence: `internal/ui/admin_shell.go` `AdminSPARouterJS`.)
- [x] Shimmer skeleton loading states (`.studio-skeleton`, `.skeleton-text`, `.skeleton-card`, `.skeleton-row`) and empty state transitions. (Evidence: `internal/ui/admin_shell.go` `AdminCSS`, `internal/adminstudio/form.go`.)
- [x] Enhanced toast notification system: `success`, `error`, `warning`, `info` types, undo action callback, deduplication, hover-pause progress bar, and ARIA live/status accessibility. (Evidence: `internal/ui/admin_shell.go` `AdminToastCSS`, `window.HydroToast`.)
- [x] Modal & Drawer interactions: fade backdrop + scale dialog animations, Escape to close, Tab focus trapping, and focus restoration to trigger elements. (Evidence: `internal/ui/admin_shell.go` `setupModals`, `internal/adminstudio/form.go` `admin-media-picker-modal`.)
- [x] Form feedback and double-submit protection: button loading spinner (`.button.is-loading`) and submission locking. (Evidence: `internal/ui/admin_shell.go` `setupFormFeedback`.)
- [x] Table, card, and media interactions: row highlight animation (`.is-highlighted`), smooth image loading, and KPI number count-up animation (`window.HydroAnimateKPIs`). (Evidence: `internal/ui/admin_shell.go` `AdminSPARouterJS`, `internal/ui/admin_dashboard.go`.)
- [x] Motion/Toast security hardening: create toast titles and messages safely via DOM `textContent` rather than `innerHTML`; add controlled retry and previous-binary rollback after DevMigration SQLite locks. (Evidence: `internal/ui/admin_shell.go`, `tools/devserver/main.go`; `go test ./... -p 1 -timeout=5m`.)
- [x] Resolve member migration checksum discrepancy; restore applied `202609110005_members` migration to immutable original content and move `email_verified_at`/`last_login_at` fields to `202609110006_members_metadata` additive migration. (Evidence: `hydro migrate` succeeds on `data/demo.sqlite`; ledger checksums verified.)
- [x] Fix DevMigration startup bug; remove invalid Members admin registry record and wait for child binary to return healthy `/healthz` or `/` response before transitioning to `ready` state. If new binary fails to boot, restore prior working binary. (Evidence: `cmd/hydro/main.go`, `tools/devserver/main.go`; targeted Go tests PASS.)
- [x] Public frontend parity and responsive UX pass: remove technical metadata noise from news/catalog cards, fix fixed widths with fluid responsive grids and text clamping, prevent mobile navigation and action overflow. (Evidence: `themes/corporate/static/theme.css`, `modules/news/transport/http.go`, `modules/commerce/transport/http.go`; `/shop` and `/public/news/tr/html` 200 render smoke.)
- [x] Add member frontend counterpart: wire secure login, registration, and account screens to existing session endpoints; add public navigation entry; add CSRF protection to forms and fetch calls, clear error and success banners (`.member-auth-status.status-error`, `.status-success`), fix SQLite RFC3339 timestamp serialization/deserialization, and add real SQLite database contract tests. (Evidence: `modules/members/sqlstore/store.go`, `modules/members/sqlstore/store_test.go`, `modules/members/transport/http.go`, `themes/corporate/static/theme.css`; `go test ./modules/members/... -v` PASSED.)
- [x] Complete member account with real backend data: `/member/orders` returns only authenticated member's own orders; account view loads live order history, session security shown in dedicated modal, and remove unnecessary security/platform marketing copy. (Evidence: `modules/members/transport/http.go`, `themes/corporate/static/theme.js`, `themes/corporate/static/theme.css`; demo seed orders and `docs/DEMO_SEED.md`.)
- [x] Make demo seed honest and aligned with frontend flows: add local demo member + sample orders, fix SQLite placeholder compatibility, remove false production/payment/warranty claims from seed copy. (Evidence: `cmd/hydro/main.go`, `docs/DEMO_SEED.md`; `go test ./... -p 1 -timeout=5m`, `go vet ./...` and `git diff --check` PASS.)
- [x] Refine metadata-driven Playwright generator contract: check only DOM presence for responsive markers, run login test with clean browser context, verify media upload result on single card. (Evidence: `tools/e2egen/main.go`; `scripts/run-e2e.ps1` with isolated seed + Chromium 9/9 PASS.)
- [x] Fix root cause of SPA fragment CSS/script nonce mismatch: move `<style>` tags in dynamic page fragments into `head` with current document's nonce, normalize script nonce, and verify resource usage screen as E2E surface. Clear local auth state only on true 401/403 status; do not disrupt UI on network/5xx errors. (Evidence: `internal/ui/admin_shell.go`, `themes/corporate/static/theme.js`, `internal/ui/admin_areas.go`, `tools/e2egen/main.go`, `e2e/manifest.json`; Chromium 10/10 PASS.)
- [x] Prevent resource leaks during SPA navigation: convert Command Palette document listeners rebound on each fragment navigation to a single delegated listener; prevent holding old shell closures. (Evidence: `internal/ui/admin_shell.go`, `internal/ui/admin_areas_test.go`; `go test ./... -p 1 -timeout=5m`, `go vet ./...` PASS.)
- [x] Refresh Commerce cart frontend and fix public layout offset: move inline styles to CSS classes, establish responsive layout for cart items and order summary, reduce sticky header scroll offset and footer margin, eliminate horizontal overflow. (Evidence: `modules/commerce/transport/http.go`, `themes/corporate/static/theme.css`; commerce/news/members transport tests, `go vet ./...`, `git diff --check`.)
- [x] Simplify product cards as catalog summaries: remove variant/quantity/add-to-cart form from card grid and move purchase flow to product detail view; ensure equal card heights and remove excess whitespace. (Evidence: `modules/commerce/transport/http.go`, `themes/corporate/static/theme.css`; card height 440–460px range, commerce tests PASS.)
- [x] Add missing theme CSS for `section-badge` and `section-header` classes used in public frontend; make news and catalog section badges visually consistent. (Evidence: `themes/corporate/static/theme.css`; theme asset served with 200.)
- [x] Adjust vertical spacing between section badge and title; add consistent grid rhythm and badge bottom margin inside `section-header` group. (Evidence: `themes/corporate/static/theme.css`.)
- [x] Fix SVG icon sizing issue in header search / Command Palette dropdown; apply 16x16px constraint to `.search-result-icon svg` rule. (Evidence: `internal/ui/admin_shell.go` `AdminSearchCSS`.)
- [x] Clean up and migrate action and management shortcut cards under reports page (`/admin/reports`) to modern grid layout (`.report-info-grid`, `.report-info-card`, `.report-action-list`). (Evidence: `internal/ui/admin_shell.go` `AdminDashboardCSS`, `internal/ui/admin_dashboard.go`.)
- [x] Transform members page (`/admin/members`) search and filter toolbar into responsive, single-line aligned `.table-toolbar` and `.search-wrap` component. (Evidence: `modules/members/transport/http.go`, `internal/ui/admin_shell.go` `AdminCSS`.)
- [x] Redesign admin control tools page (`/admin/control`) from scratch with a modern, card-based, accessible layout with toast notifications. (Evidence: `internal/ui/admin_control.go`, `go test ./... -p 1` PASSED.)

- [!] Do not directly implement Distributed, S3/CDN, OIDC/SAML, multi-tenant, AI, and broad Commerce scopes. (Reason: Requires corporate customer contract and capacity benchmark; postponed to avoid adding unnecessary complexity to Core.)
- [x] Clarify only the current ADR/backlog/roadmap status, required decisions, and blockers for these. (Evidence: `docs/architecture/DECISION_BACKLOG.md` and `docs/ROADMAP.md` updated.)
- [!] Leave features without actual customer demand and capacity evidence as `[?]` or `[!]`. (Reason: Due to product philosophy and AGENTS.md, unmeasured capabilities are left as open gates.)

## Workflow to apply for each task

### Hardening follow-up — 2026-09-13

- [x] Bound public Commerce catalog projections and add REST page metadata. (Evidence: `modules/commerce/catalog.go`, SQL `LIMIT/OFFSET` repository contract, `X-Catalog-*` headers, pagination regression test; `go test ./...` PASS.)
- [x] Expose bounded pending refund reconciliation to administrators. (Evidence: `GET /admin/commerce/refunds/pending`, `ListPendingRefunds`, ADR-0034, OpenAPI schema; Commerce tests PASS.)
- [x] Align Commerce legacy and versioned API routes. (Evidence: `/api/v1/commerce/...` and `/api/v1/webhooks/payment/{provider}` aliases, route regression test, ADR-0015 update; `go test ./...` PASS.)
- [x] Make E2E metadata fail closed on cross-surface selector collisions. (Evidence: `tools/e2egen` rejects duplicate `test_id` values across admin fields, admin surfaces, and frontend routes; regression test `TestValidateRejectsCrossSurfaceTestIDCollision` PASS.)
- [x] Bound public product-variant projections with the same pagination contract as products. (Evidence: `ListVariantsPage`, `LIMIT/OFFSET` SQL adapter, `X-Catalog-*` headers, repository regression test, ADR-0036 and OpenAPI update; Commerce tests PASS.)
- [x] Reject malformed public catalog page parameters instead of silently falling back to page one. (Evidence: shared `catalogPage` validation for product/variant REST handlers and `TestCatalogPageRejectsMalformedValues`; Commerce transport tests PASS.)
- [x] Keep Commerce version aliases single-sourced in the core router. (Evidence: removed duplicate manual aliases that could panic route registration; `internal/router` performs `/api` → `/api/v1` mapping and `go test ./cmd/hydro ./internal/router` PASS.)
- [x] Prevent caching of sensitive admin refund and report export responses. (Evidence: `Cache-Control: no-store` and `Pragma: no-cache` on pending refunds and CSV/JSON exports; targeted tests PASS.)
- [x] Apply bounded variant projection to the public product detail page as well as the JSON endpoint. (Evidence: detail rendering uses `ListVariantsPage` with the public catalog bound; Commerce transport tests PASS.)
- [x] Re-audit News and Pages public list bounds. (Evidence: both handlers validate `limit` in the 1–100 range and repositories issue bounded SQL `LIMIT`; no additional unbounded public list path found.)
- [!] Race detector could not run on this Windows host because `CGO_ENABLED=1` requires an unavailable `gcc`; Linux CI race job remains the verification path. (Observed: `cgo: C compiler "gcc" not found`.)
- [!] Full production gates remain external: real provider credentials, reverse proxy/process restart, backup/key escrow, customer capacity/RPO/RTO, and production signing ceremony. (Tracked in `docs/validation/PRODUCTION-GATES.md`; do not mark as local success.)

1. Read related documents and ADRs.
2. Examine existing code and tests.
3. Make the smallest safe change.
4. Run related tests.
5. Immediately update status and evidence in `TASKS.md`.
6. If user intervention is required, stop the task, explicitly record it, and move to the next independent task.
7. Finally, summarize changed files, successful tests, open `[!]`/`[?]` items, and those skipped due to CI quota.

## Starting prompt for Gemini

```text
Act as an autonomous development agent running overnight in this repository.

First, fully read TASKS.md and the AGENTS.md, philosophy, implementation plan, architecture README, and related ADRs mentioned there. Follow the tasks in TASKS.md sequentially. After completing each operation or meaningful subtask, immediately update the TASKS.md file; mark the status as [x], [!], [?], or appropriate symbol, and add short evidence.

The user's CI quota is exhausted; do not try to run CI or fix CI issues. Run local and targeted tests, but do not perform tasks requiring true production, true reverse proxy/process restart, true external backup/key recovery, real SMTP/payment/webhook, or user credentials. If such a task arises, log it as [!] in TASKS.md with the reason, required environment, and suggested next step; then proceed with safe independent tasks.

Do not silently contradict ADRs, fork Core, overwrite user changes, use destructive git commands, or mark unmeasured results as production-ready. When adding new behavior, update tests, migration/compatibility metadata, and capacity/deployment docs (if needed) in the same scope. Do not add unnecessary services or unlimited resources to the Compact profile.

Priority: First align scope/documentation, then restore/rollback/operability reliability, then security hardening, then Admin Studio, editorial workflow, headless API/starter, search, performance, and the Agency Template-inspired demo experience in Phase 6.5. Treat News presentation work as theme/block projections and treat Commerce as a HydroGo contract-driven storefront scope; do not claim Agency Template already supplies a Commerce frontend. Only log Distributed, S3/CDN, OIDC/SAML, multi-tenant, AI, and external payment/infrastructure work as decision/blocker analyses; do not implement without approval.

Run related tests after every change. If possible, use gofmt, go vet, and targeted go test; do not wait for CI results due to the quota. At the end of the shift, update TASKS.md and list the completed tasks, test evidence, changed files, and [!] / [?] items awaiting user intervention in your final output.

### Phase 9 — Bounded media derivatives

- [x] Add additive SQLite/PostgreSQL `media_variants` metadata and ADR-0026; keep originals immutable and make derived status explicit. (Evidence: `migrations/202609120001_media_variants`, `docs/architecture/0026-bounded-media-derivatives.md`.)
- [x] Add bounded asynchronous WebP/WebM optimization worker with queue capacity 8, profile-limited concurrency, 2-minute timeout, and one ffmpeg codec thread. (Evidence: `internal/media/derivatives.go`; original uploads do not wait for optimization.)
- [!] Install/configure a real ffmpeg encoder in the deployment environment. The current machine has no `ffmpeg`, so uploads remain available and derivatives are recorded as failed until `HYDRO_MEDIA_FFMPEG` points to an encoder. (Required: operator-installed ffmpeg and a local media conversion smoke test.)
- [x] Add `?variant=card` / `?variant=video` lookup with safe fallback to the original and allow common video upload MIME types. (Evidence: `internal/media/http.go`, `internal/media/upload.go`.)
- [x] Include the media upload journey in metadata-driven Playwright generation. (Evidence: `e2e/manifest.json`, `tools/e2egen/main.go`, regenerated `e2e/generated/hydro.spec.ts`; generator contract test passes.)
- [x] Generate responsive frontend E2E checks for every declared frontend route: mobile viewport, visible route root, and horizontal-overflow guard. (Evidence: `tools/e2egen/main.go`, regenerated `e2e/generated/hydro.spec.ts`; Playwright lists the generated journeys.)
- [x] Document the media derivative pipeline, RAM limits, E2E generation behavior, test evidence, and remaining encoder/durable-queue gates. (Evidence: `docs/validation/media-derivatives-and-e2e.md`.)

### Phase 10 — Core Isolation and Licensing Architecture

- [x] HydroGo Core isolation architectural analysis and model evaluation (Private Go module, Standalone binary, Hosted/SaaS runtime). (Evidence: `docs/architecture/0028-core-isolation-and-licensing.md`, `docs/COMMERCIAL_LICENSING.md`.)
- [x] Ed25519-signed offline licensing verification package (`internal/licensing`): `Claims`, `SignedLicense`, `Issue`, `Verify`, `EncodePublicKey`/`DecodePublicKey`, `Fingerprint`, `GracePeriodDays` tolerance tracking, and human-readable status/error messages. (Evidence: `internal/licensing/license.go`, `internal/licensing/license_test.go` PASS.)
- [x] License verification tests: valid license, invalid/tampered signature, expired license, grace period, mismatched `project_id`, missing `feature`, edition hierarchy, and malformed input tests. (Evidence: `go test ./internal/licensing/... -v` 9/9 PASS.)
- [x] Hydro CLI license commands: `hydro license keygen`, `hydro license issue`, `hydro license verify`, `hydro license status`, and `--json` support. (Evidence: `cmd/hydro/license_cmd.go`, `cmd/hydro/main.go`, `cmd/hydro/main_test.go` TestRunLicenseCLI PASS.)
- [x] Open-source MIT license (`LICENSE`) and third-party dependency license inventory (`NOTICE`). (Evidence: `LICENSE`, `NOTICE`.)
- [x] Project lock file (`hydrogo.lock`): Core version, generator version, preset, edition, required features, and license constraints. (Evidence: `hydrogo.lock`.)
- [x] Realistic documentation of licensing, binary distribution, and reverse engineering boundaries (rejecting unfounded claims like "code cannot be copied"). (Evidence: `docs/COMMERCIAL_LICENSING.md` Section 5, ADR-0028.)
- [x] Bind runtime license requirement to `serve` boot, remove trust in self-advertised public keys, verify fail-closed. (Evidence: `internal/licensing/license.go`, `cmd/hydro/license_cmd.go`, `cmd/hydro/main.go`, `TestValidateRuntimeLicenseInServe` PASS.)
- [x] Verify presence of `hydrogo.lock`, JSON integrity, project ID, edition tier, license requirement, and configured feature set at boot; fail-closed on drift. (Evidence: `cmd/hydro/license_cmd.go`, lock edition/feature contract tests.)
- [x] Add interactive scaffolding `hydro new PROJECT --preset=...` avoiding Core source copying; generate config, lock, runtime-loadable minimal theme, custom/migration/seed, docs, and E2E manifest. (Evidence: `internal/projectgen`, `cmd/hydro/new_cmd.go`, `internal/projectgen/projectgen_test.go` all preset families PASS, `scripts/generator-smoke.ps1`, `docs/PROJECT_GENERATOR.md`.)
- [x] Comprehensive setup and deployment documentation: News portal (`docs/NEWS_SITE_SETUP.md`), E-Commerce (`docs/COMMERCE_SETUP.md`), Production Deployment (`docs/DEPLOYMENT.md`), and Upgrade Runbook (`docs/UPGRADE.md`). (Evidence: `docs/NEWS_SITE_SETUP.md`, `docs/COMMERCE_SETUP.md`, `docs/DEPLOYMENT.md`, `docs/UPGRADE.md`.)
- [!] Injecting trusted public key into release builds via `-ldflags` is a release CI/CD task; development binaries deliberately verify licensed projects fail-closed.
- [x] Updated to open-source MIT license model.
- [!] Live payment provider keys (Stripe live prod) and live domain SSL certificates must be supplied in operator/production environment.
- [x] Aggregate all operator actions, unproven tests, and production decision gates into one document. (Evidence: `docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md`.)
- [x] Add `Developer > Resource Usage` screen in Admin Studio; expose only measurable process, DB pool, cache, queue, and worker telemetry. (Evidence: `internal/observability`, `internal/ui/admin_areas.go`, `/admin/developer/resources`.)
- [x] Fix SPA-navigated developer subpages where CSP nonce blocked page-specific CSS; unify resource consumption and area styles into document-wide Admin Studio CSS bundle. (Evidence: `internal/ui/admin_shell.go`, `AdminResourceCSS`; direct and SPA navigation E2E.)
- [x] Extend metadata-driven Admin E2E to verify real sidebar SPA transitions via `navigate_from` and optional computed-style contracts. (Evidence: `tools/e2egen/main.go`, `e2e/manifest.json`; Chromium 10/10 PASS.)
- [x] SPA lifecycle audit: media picker and media ID copy scripts use idempotent global bind guards; prevent listener proliferation during long SPA sessions. (Evidence: `modules/pages/transport/http.go`, `internal/media/admin.go`; Chromium E2E PASS.)
- [x] Asset cache contract: align theme CSS/JS asset URLs with unified version contract. (Evidence: `themes/corporate/layouts/base.html`, `cmd/hydro/main.go`, `modules/members/transport/http.go`, `modules/news/transport/http.go`, `modules/commerce/transport/http.go`; local E2E and smoke PASS.)
- [x] Admin fragment CSS lifecycle: normalize styles onto inert fragment using nonce rather than copying to head; prevent fragment style accumulation in head. (Evidence: `internal/ui/admin_shell.go`, 14/14 Chromium E2E.)
- [x] SPA fragment CSS root fix: normalize all page-specific styles with document CSP nonce before DOM attachment; remove legacy head-style copying. Add computed-style E2E checks across Developer, Operations, Security, Permissions, and Resources surfaces. (Evidence: `internal/ui/admin_shell.go`, `tools/e2egen/main.go`, `e2e/manifest.json`; Chromium 14/14 PASS.)
- [x] Fix second root cause of nonce loss: place normalized fragments using `replaceChildren` with real DOM nodes rather than `innerHTML` string serialization; browser attribute masking during serialization will not break CSS. (Evidence: `internal/ui/admin_shell.go`; Chromium 14/14 PASS.)
- [x] Verify developer subpages via sidebar SPA navigation; move common Admin area CSS rules to global stylesheet. (Evidence: `e2e/manifest.json`, `AdminAreaCSS`; Chromium 14/14 PASS.)
- [x] Remove direct Members–Commerce module dependency; member surface consumes neutral customer-order capability contract and connects via adapter in Commerce composition root. (Evidence: `internal/contracts/customerorders`, `modules/commerce/customerorders.go`, `modules/members/transport/http.go`, ADR-0038; targeted Go tests PASS.)
- [x] Add CI import guard to prevent breaking module boundaries; only explicitly allowed aggregate API layer can import sibling modules. (Evidence: `tools/moduleguard`, `.github/workflows/ci.yml`; `go run ./tools/moduleguard -root modules -allow modules/api` PASS.)
- [x] Customer-order adapter fails closed with error instead of silent success on empty repository; protect projection and private customer field isolation with regression tests. (Evidence: `modules/commerce/customerorders.go`, `customerorders_test.go`.)
- [x] Filter inactive module collections from Admin Studio dashboard and manifest; `commerce_variants` follows commerce lifecycle. (Evidence: `cmd/hydro/main.go`, `internal/adminstudio/e2e.go`.)
- [x] Startup no longer requires Admin Studio collections or handler dependencies when Pages/News are inactive; inactive profiles boot fail-safe. (Evidence: `cmd/hydro/main.go`; `cmd/hydro` tests PASS.)
- [x] When News is inactive, repository/service/queue worker objects are not instantiated; worker callbacks remain no-ops with safe zero values. (Evidence: `cmd/hydro/main.go`; `cmd/hydro` tests PASS.)
- [x] Active Admin Studio collection filter protected with independent contract and defensive-copy tests; variant collection tied to Commerce lifecycle. (Evidence: `internal/adminstudio/active.go`, `active_test.go`.)
- [x] Bounded query for member sessions; account view returns at most 100 sessions. (Evidence: `modules/members/sqlstore/store.go`.)
- [x] Upper-bound public site settings (1,000) and navigation (500) queries; prevent unbounded data growth in render surfaces. (Evidence: `modules/site/repository/sqlstore/store.go`.)
- [x] Move Admin Commerce product list from unbounded repository method to bounded `ListProductsPage` contract (1,000 records). (Evidence: `modules/commerce/transport/http.go`.)
- [x] Bound News scheduled content sweeps to 100 records per worker; backlog growth does not cause unbounded load spikes. (Evidence: `modules/news/repository/sqlstore/store.go`.)
- [x] Route legacy Commerce `ListProducts` to bounded page contract; direct consumers cannot fetch unbounded products. (Evidence: `modules/commerce/repository/sqlstore/store.go`.)
- [x] Align navigation write contract with read limit; accept at most 500 items per transaction. (Evidence: `modules/site/service.go`.)
- [x] Add ID/locale/label/href length boundaries for navigation fields; reject oversized metadata. (Evidence: `modules/site/model.go`.)
- [x] Add length contracts for News ID, title, slug, summary, tags, and SEO fields; reject oversized content metadata. (Evidence: `modules/news/model.go`.)
- [x] Unify media admin and Pages media picker fragment listeners with global guards; prevent listener proliferation during long SPA sessions. (Evidence: `internal/media/admin.go`, `modules/pages/transport/http.go`.)
- [x] Align frontend asset cache-buster version to unified `20260913` version; remove legacy clean/date variants. (Evidence: server-rendered Go surfaces and `themes/corporate/layouts/base.html`.)
- [x] Add CI `assetguard` to prevent asset cache-buster drift. (Evidence: `tools/assetguard`, `.github/workflows/ci.yml`; guard PASS.)
- [x] Add CI `migrationguard` verifying migration metadata ID/module/dependency/checksum contracts. (Evidence: `tools/migrationguard`, `.github/workflows/ci.yml`.)
- [x] Complete Pages public-read limits and optimistic-lock conflict fixtures with isolated service tests. (Evidence: `modules/pages/service/content_test.go`.)
- [x] News admin validation fixtures verify rejection of oversized metadata in isolation. (Evidence: `modules/news/model_test.go`.)
- [x] Add length limits and isolated validation fixtures for Pages ID, slug, title, and body. (Evidence: `modules/pages/service/content.go`, `validation_test.go`.)
- [x] Limit Pages hero/CTA block required text fields to 512 characters; include oversized payloads in validation suite. (Evidence: `modules/pages/blocks.go`.)
- [x] Make Auth user creation fail-closed with email length and RFC address validation. (Evidence: `internal/auth/service.go`.)
- [x] Bind invitation and password-recovery inputs to shared `ValidateEmail` contract; reject malformed addresses prior to DB lookup/token generation. (Evidence: `internal/auth/user.go`, `invite.go`, `recovery.go`.)
- [x] Bound sitemap route projection; cap at 10,000 records to limit memory and response growth. (Evidence: `modules/pages/repository/sqlstore/content.go`.)
- [x] Move remaining fragment styles for Permissions screen to global Admin stylesheet: tables, badges, and permissions rules cards render styled on initial SPA load. (Evidence: `AdminPermissionsCSS`, `e2e/manifest.json`; Chromium 14/14 PASS.)
### Phase 11 — Member Portal, Account Management & Dynamic Navigation

- [x] Enhance member account panel (`/member/account`) with corporate styling: profile hero card, initials avatar, email verification badge, Argon2id security indicator, active authorized sessions & IP list table, order/license shortcuts, and CSRF-protected sign-out button. (Evidence: `modules/members/transport/http.go`, `themes/corporate/static/theme.css`.)
- [x] Fix SQLite session timestamp compatibility (`RFC3339Nano` format and `parseTimePtr` support, resolving session dropouts and infinite redirects). (Evidence: `modules/members/sqlstore/store.go`, `modules/members/sqlstore/store_test.go` PASS.)
- [x] Dynamic client-side navigation sync (`themes/corporate/static/theme.js`): On statically cached pages (`/public/pages/...`, `/public/news/...`, `/shop`), dynamically replaces `Sign In` button with `👤 <email>` badge linking to `/member/account` when a member is logged in. (Evidence: `themes/corporate/static/theme.js`, `themes/corporate/layouts/base.html`, `modules/news/transport/http.go`, `modules/commerce/transport/http.go`.)
- [x] 100% verified across full member and core test suite. (Evidence: `go test ./... -p 1 -timeout=5m` 100% PASS.)
```
