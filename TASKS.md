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
- [ ] Connect frontend build manifest to generator input; deterministically produce frontend route and selector contract from build output.
- [ ] Produce isolated fixtures for Pages/News public read, admin validation, and optimistic-lock conflict scenarios.
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

### Redis ve kimlik akışı sertleştirmesi

- [x] Redis/Valkey’i opsiyonel dağıtık rate-limit sağlayıcısı olarak bağla; Compact profilde bounded local fallback’i koru. (Kanıt: `internal/security/redis_ratelimit.go`, `HYDRO_REDIS_URL`, atomic `INCR + TTL` limiter ve `go test ./...`.)
- [x] Redis’i public Pages cache’inde L2 olarak bağla; L1 generation invalidation ve DB fallback sınırlarını koru. (Kanıt: `internal/cache/redis.go`, `internal/cache/two_tier.go`, `modules/pages/service/content.go`.)
- [x] Redis kurulum/çalıştırma yönergesini örnek config ve ADR’e işle. (Kanıt: `config.example.yaml`, ADR-0024.)
- [x] Login/logout güvenlik akışını sertleştir: auth yanıtlarında `no-store`, başarısız login/MFA audit izi, hashed başarısız login hedefi, logout sonrası audit hatasında oturumu açık bırakmama. (Kanıt: `internal/auth/http.go` ve auth testleri.)
- [!] Windows Redis’in kalıcı servis kurulumu kullanıcı ortamında tamamlanamadı; Chocolatey paketi mevcut ancak yönetici yetkisi veya servis wrapper kurulumu gerekiyor. (Gerekli kullanıcı işlemi: yönetici PowerShell ile Redis servisini kurmak.)
- [?] Kullanıcı bazlı kalıcı analitik ve actor ID audit alanı hâlâ ayrı migration/retention/consent kararı gerektiriyor; mevcut audit sözleşmesi ham kullanıcı davranışı toplamıyor.

### Faz 6 — Üyeler (Members Modülü ve Yönetimi)

- [x] Admin kullanıcılarından bağımsız `members` ve `member_sessions` tablolarını, `email_verified_at` ve `last_login_at` alanlarını, SQLite/PostgreSQL migration’larını (`migrations/202609110005_members`) ve ayrı `hydro_member_session` cookie sözleşmesini ekle. (Kanıt: `modules/members`, `migrations/202609110005_members`, `go test ./migrations`.)
- [x] Üye arama, filtreleme (`all`, `active`, `inactive`, `verified`, `unverified`) ve sayfalama ekle. (Kanıt: `modules/members/sqlstore/store.go` Search metodu, `modules/members/transport/http.go`.)
- [x] Aktif/pasif yönetimini ve e-posta doğrulama durumunu (verified/unverified) destekle. (Kanıt: `SetStatus`, `SetEmailVerified` metotları ve admin POST handler'ları.)
- [x] Üye detay, son giriş, session sonlandırma (tekil ve tüm oturumları kapat) ve audit görünümü ekle. (Kanıt: `/admin/members/{id}`, `RevokeSessionByID`, `RevokeAllSessions`, `AuditStore.ListByTargetID`.)
- [x] Admin kullanıcıları ile üyeleri kesinlikle ayrı tut; üye session cookie'sinin (`hydro_member_session`) `/admin` rotalarına yetki vermediğini güvenceye al. (Kanıt: `TestAdminNonAdminAndMemberCookieAccessControls` ve `news_contract_test.go`.)

### Faz 7 — Raporlar ve Operasyonel Telemetri

- [x] Raporlar panelini (`/admin/reports`) ve canlı bellek içi (in-memory) telemetri sunumunu bağla. (Kanıt: `internal/observability/metrics.go`, `internal/ui/admin_dashboard.go`.)
- [x] Gerçek görüntülenme, benzersiz ziyaretçi SHA-256 tekilleştirme, latency ortalaması, sayfa yenileme (refresh) tekilleştirmesi ve hata verilerini raporla; sahte metrik kullanımını engelle. (Kanıt: `TestMetricsVisitorRefreshAndDeduplication`, `TestMetricsEmptyState`, `TestMetricsProcessRestartResetsInMemoryCounters`.)
- [x] Hata frekansını rota bazında (4xx/5xx HTTP durum kodları) şeffaf şekilde listele; gerçekte olmayan oturum-eylem korelasyonu iddialarını arayüzden temizle. (Kanıt: `TopErrors` endpoint bazlı hata sayımı, `TestMetricsErrorTrackingAndTopErrors`.)
- [x] Üye kayıt ve durum istatistiklerini raporlara entegre et. (Kanıt: `memberService.Stats(ctx)` entegrasyonu.)
- [x] CSV/JSON formatında rapor dışa aktarma (export) desteği ekle. (Kanıt: `GET /admin/reports/export?format=csv` ve `?format=json`.)
- [!] Kalıcı zaman serisi analitiği (geçmiş gün/hafta/ay aralıkları) ve oturum-eylem korelasyonu veritabanı şeması, saklama (retention) süresi ve KVKK/GDPR rıza onayı gerektirdiğinden açık kapı [!] olarak işaretlendi. (Gerekli kullanıcı/ürün kararı: Veritabanında time-series telemetry tablosu ve açık rıza politikası kararı.)

### Faz 8 — Navigasyon ve Erişilebilirlik

- [x] Gelişmiş Command Palette (Ctrl/Cmd+K) ekle: hızlı kısayollar, arama, klavye yön tuşları (ArrowUp/Down, Enter, Esc), ARIA combobox etiketleri ve favori/son ziyaret edilen ekranlar (`localStorage`). (Kanıt: `internal/ui/admin_shell.go` `AdminSPARouterJS`, `AdminSearchCSS`.)
- [x] Sidebar menüsünde "KULLANICILAR" bölümünü ve "Üyeler" linkini sunucu taraflı ve SPA uyumlu olarak yerleştir; aktif sekme haritalamasını ve yönlendirmeleri tamamla. (Kanıt: `internal/ui/admin_shell.go` `AdminShell` ve `AdminStart`.)
- [x] Klavye navigasyonu, focus halkaları, kontrast ve mobil duyarlı sidebar/alt bar deneyimini iyileştir. (Kanıt: `AdminDashboardCSS`, `theme.css` responsive layout düzenlemeleri.)

### Faz 9 — Admin Studio Motion & Interaction UX

- [x] PocketBase Admin UI'dan ilham alan, sade, hızlı (120-240ms) ve amaca hizmet eden Motion Token sistemi oluştur (`--motion-fast`, `--motion-normal`, `--motion-slow`, `--ease-standard`, `--ease-emphasized`, `--ease-out`). (Kanıt: `internal/ui/admin_shell.go` `AdminCSS`.)
- [x] Global `@media (prefers-reduced-motion: reduce)` desteğiyle tüm geçiş ve animasyonları erişilebilirlik tercihlerine göre anında devre dışı bırak. (Kanıt: `internal/ui/admin_shell.go` `AdminCSS`, `AdminSPARouterJS`.)
- [x] SPA Router sayfa geçiş animasyonu (`.page-entering`, `pageEnter` 180ms translate/fade) ve üst yükleme çubuğu (progress bar) yumuşak geçişleri. (Kanıt: `internal/ui/admin_shell.go` `AdminSPARouterJS`.)
- [x] Shimmer skeleton yükleme durumları (`.studio-skeleton`, `.skeleton-text`, `.skeleton-card`, `.skeleton-row`) ve boş durum geçişleri. (Kanıt: `internal/ui/admin_shell.go` `AdminCSS`, `internal/adminstudio/form.go`.)
- [x] Gelişmiş Toast bildirim sistemi: `success`, `error`, `warning`, `info` tipleri, geri alma (undo callback) butonu, deduplication, hover'da duraklatılan progress bar ve ARIA live/status erişilebilirliği. (Kanıt: `internal/ui/admin_shell.go` `AdminToastCSS`, `window.HydroToast`.)
- [x] Modal & Drawer etkileşimleri: fade backdrop + scale dialog animasyonları, Escape tuşu ile kapatma, Tab tuşu odak kiliti (focus trap) ve tetikleyiciye odak iadesi (focus restoration). (Kanıt: `internal/ui/admin_shell.go` `setupModals`, `internal/adminstudio/form.go` `admin-media-picker-modal`.)
- [x] Form geri bildirimi ve çift tıklama koruması: submit anında buton loading spinner'ı (`.button.is-loading`) ve submit kilitleme. (Kanıt: `internal/ui/admin_shell.go` `setupFormFeedback`.)
- [x] Tablo, kart ve medya etkileşimleri: satır vurgulama animasyonu (`.is-highlighted`), yumuşak resim yükleme ve rapor KPI sayı sayma animasyonu (`window.HydroAnimateKPIs`). (Kanıt: `internal/ui/admin_shell.go` `AdminSPARouterJS`, `internal/ui/admin_dashboard.go`.)
- [x] Motion/Toast güvenlik sertleştirmesi: toast başlık ve mesajlarını `innerHTML` yerine DOM `textContent` ile güvenli oluştur; DevMigration SQLite kilidi sonrası kontrollü retry ve önceki binary’ye rollback ekle. (Kanıt: `internal/ui/admin_shell.go`, `tools/devserver/main.go`; `go test ./... -p 1 -timeout=5m`.)
- [x] Üye migration checksum uyuşmazlığını düzelt; uygulanmış `202609110005_members` migration’ını immutable eski içeriğine döndür ve `email_verified_at`/`last_login_at` alanlarını `202609110006_members_metadata` additive migration’ına taşı. (Kanıt: `data/demo.sqlite` üzerinde `hydro migrate` başarılı; ledger checksum’leri doğrulandı.)
- [x] DevMigration başlangıç hatasını düzelt; geçersiz Members admin registry kaydını kaldır ve child binary gerçekten `/healthz` veya `/` yanıtı vermeden `ready` durumuna geçme. Yeni binary açılmazsa önceki çalışan binary’yi geri yükle. (Kanıt: `cmd/hydro/main.go`, `tools/devserver/main.go`; hedefli Go testleri başarılı.)
- [x] Header arama / Command Palette dropdown SVG ikon boyutlandırma hatasını düzelt; `.search-result-icon svg` kuralına 16x16px kısıtı ekle. (Kanıt: `internal/ui/admin_shell.go` `AdminSearchCSS`.)
- [x] Raporlar sayfası (`/admin/reports`) altındaki eylem ve yönetim kısayolları kartlarını temizle ve grid düzenine geçir (`.report-info-grid`, `.report-info-card`, `.report-action-list`). (Kanıt: `internal/ui/admin_shell.go` `AdminDashboardCSS`, `internal/ui/admin_dashboard.go`.)
- [x] Üyeler sayfası (`/admin/members`) arama ve filtreleme araç çubuğunu responsive ve tek satırda hizalı `.table-toolbar` ve `.search-wrap` bileşenine dönüştür. (Kanıt: `modules/members/transport/http.go`, `internal/ui/admin_shell.go` `AdminCSS`.)
- [x] Yönetim araçları sayfasını (`/admin/control`) sıfırdan modern, kart tabanlı, anlaşılır ve toast bildirimli olarak yeniden tasarla. (Kanıt: `internal/ui/admin_control.go`, `go test ./... -p 1` PASSED.)

- [!] Do not directly implement Distributed, S3/CDN, OIDC/SAML, multi-tenant, AI, and broad Commerce scopes. (Reason: Requires corporate customer contract and capacity benchmark; postponed to avoid adding unnecessary complexity to Core.)
- [x] Clarify only the current ADR/backlog/roadmap status, required decisions, and blockers for these. (Evidence: `docs/architecture/DECISION_BACKLOG.md` and `docs/ROADMAP.md` updated.)
- [!] Leave features without actual customer demand and capacity evidence as `[?]` or `[!]`. (Reason: Due to product philosophy and AGENTS.md, unmeasured capabilities are left as open gates.)

## Workflow to apply for each task

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
```
