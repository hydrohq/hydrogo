# HydroGo uygulama durumu ve çalışma rehberi

Bu belge, planlanan işlerle gerçekten teslim edilmiş işleri ayırır. Bir özelliğin
burada “tamamlandı” olması, kodun repoda bulunması ve yerel testinin geçmesi
anlamına gelir; production hazır olduğu anlamına gelmez.

Blueprint Admin Studio'nun müşteri deneyimi, Super Admin sınırı, otomatik
aktivasyon ve doğrulama boşlukları için güncel denetim ve aşamalı teslim planı:
[`docs/planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md`](planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md).

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
Clean scaffold to native first CRUD remains unmeasured because the project
scaffold needs the separately licensed runtime; existing measurements are not
presented as equivalent. See the Admin Studio audit plan.

## Tamamlanan ve doğrulanan parçalar

| Alan | Durum | Nasıl kullanılır? | Kanıt |
| --- | --- | --- | --- |
| Consumer/member mobile API | Tamamlandı (yerel) | Ayrı `members.Member` bearer oturumu ile kayıt/giriş/refresh, cihaz iptali, profil, sepet, checkout, sipariş listesi ve sahiplik-kapsamlı ödeme durumu | ADR-0150, `R66-release-gate-manifest-2026-09-19`, `R69-member-order-detail-payment-status-2026-09-19`, `docs/api/member-mobile-v1.openapi.json`, `docs/api/member-commerce-v1.openapi.json` |
| Public proje komut alanı | Tamamlandı | `hydro project new`, `hydro project check` (varsa generated resource'ları da dependency-free doğrular) | `R5-sdk-project-namespace`, `R5-sdk-project-check`, `R6-project-resource-preflight` |
| Payload uyumlu CLI akışı | Tamamlandı | `check`, `module new`, `resource new`, `resource check` ve `api diff`; eski komutlar korunur | `R5-cli-new-aliases` |
| İmzalı schema/migration diff | Tamamlandı | Release ledger checksum karşılaştırması; silme/değişiklik fail-closed | `R5-schema-diff` |
| Permission contract preflight | Tamamlandı | Güvenli/tekrarsız permission adları ve `permissions check` | `R5-permissions-check` |
| Üretilen SDK rehberi | Tamamlandı | Yeni projede `docs/SDK.md` | `R5-generated-sdk-guide` |
| Runtime artifact doğrulama | Tamamlandı | `hydro runtime verify [--root PATH]` | `R5-runtime-artifact-verification` |
| Signed manifest artifact hygiene | Tamamlandı | Duplicate/path/size/checksum fail-closed kontrolleri | `R5-release-manifest-artifact-hygiene` |
| Platform hedefi ve artifact çözümleme | Tamamlandı | `linux-amd64`, `windows-arm64` vb. hedefler | `R5-sdk-platform-targets`, `R5-sdk-artifact-resolution` |
| Güvenli runtime cache yolu | Tamamlandı | Ürün/sürüm/hedef bazlı deterministik cache | `R5-sdk-runtime-cache` |
| Cache listeleme/temizleme | Tamamlandı | Sürüm koruma listesi + dry-run + symlink reddi | `R5-sdk-cache-gc` |
| Atomik artifact kurulumu | Tamamlandı | checksum/size doğrulaması + atomic rename | `R5-sdk-atomic-install` |
| Download URL ve bounded download | Tamamlandı | HTTPS, redirect kapalı, boyut sınırı | `R5-sdk-download-url-policy`, `R5-sdk-bounded-download` |
| Grant doğrulama ve indirme bağı | Tamamlandı | proje/core/sdk/expiry/url kontrolleri | `R5-sdk-artifact-grant`, `R5-sdk-grant-download-integration`, `R5-sdk-grant-parser` |
| Broker grant istemcisi | Tamamlandı (yerel) | Bounded POST, koordinat/feature preflight, token header, redirect/status/response ve tek çağrıda lock doğrulaması | `R5-sdk-broker-client` |
| Typed frontend query adapter | Kısmi tamamlandı (yerel) | Üretilen `react.ts` içinde framework-neutral typed query options ve stabil query key yardımcıları; generated API client artık bounded query ve AbortSignal kullanıyor | `R5-typed-frontend-query-adapter`, `R5-generated-client-query-cancellation` |
| Taxonomy contract | Kısmi tamamlandı (yerel) | Çok dilli kategori/etiket, parent cycle ve related-content referanslarını bounded doğrulayan provider-neutral sözleşme | `R5-taxonomy-contract` |
| Search contract | Kısmi tamamlandı (yerel) | Bounded query/locale/pagination ve deterministik sonuç sözleşmesi; provider/index seçimi açık | `R5-search-contract` |
| Generated SWR adapter | Kısmi tamamlandı (yerel) | Kaynaksız dependency-free typed key/fetcher çıktısı; browser entegrasyonu açık; tüm adapter'lar için namespaced `index.ts` giriş noktası üretildi | `R5-generated-swr-adapter`, `R5-generated-frontend-index` |
| Generated frontend query bounds | Tamamlandı (yerel) | Üretilen query adapter page/page-size/sort/filter sınırlarını istek URL'si oluşmadan uygular; browser/provider kabulü açık | `R5-generated-query-input-bounds` |
| Form contract | Kısmi tamamlandı (yerel) | Bounded metadata ve submission normalizasyonu; inbox/consent/delivery modüle ait | `R5-form-contract` |
| Kit dependency resolution | Kısmi tamamlandı (yerel) | Allowlisted dependency expansion, validated registry constructor ve deterministic kit set; generated admin/SDK/runtime artifacts now include resolved dependencies and typed optional descriptors; implementation loading/compatibility matrix modüle ait | `R5-kit-dependency-resolution`, `R5-generated-kit-dependency-artifacts`, `R5-kit-descriptor-metadata`, `R6-validated-kit-registry` |
| Generated form schema | Kısmi tamamlandı (yerel) | Collection spec’inden deterministic `form.schema.json`; business validation/inbox modüle ait | `R5-generated-form-schema` |
| Frontend starter manifest | Kısmi tamamlandı (yerel) | Preset/presentation/features/design-system seçeneklerini bildiren credential-free manifest | `R5-frontend-starter-manifest` |
| Frontend manifest CLI check | Tamamlandı (yerel) | `hydro frontend check --manifest PATH [--json]` proje üretmeden aynı fail-closed route doğrulamasını çalıştırır | `R5-frontend-manifest-check-command` |
| Frontend manifest input bounds | Tamamlandı (yerel) | Manifest dosyası 1 MiB, route adı/path/test-id uzunlukları bounded ve non-regular dosyalar reddedilir | `R5-bounded-frontend-manifest-input` |
| Generated resource self-description | Tamamlandı (yerel) | Her generated collection deterministic `spec.json` ve artifact checksum'ları taşır; `hydro resource check --root` spec dosyalarını keşfedip tüm scaffold'u dependency-free ve fail-closed doğrular | `R6-generated-resource-discovery`, `R6-generated-artifact-integrity` |
| Generated OpenAPI query/error contract | Tamamlandı (yerel) | OpenAPI çıktısı bounded list query parametrelerini ve standart `HydroAPIError` yanıtlarını içerir; `hydro validate api` request parametre/body silinmesi ve optional→required kırılmalarını da yakalar | `R6-generated-openapi-contract`, `ADR-0026` |
| Versioned global resource contract | Tamamlandı (yerel) | Blueprint `type: global` tekil kaynak metadata'sı, global ID normalizasyonu ve admin manifest/API sözleşmesi üretir | `ADR-0224`, `305072e1`, `c5f702df` |
| Generated global singleton transport | Tamamlandı (yerel) | Global kaynakların boş route GET'i stable `global` ID ile tekil kayda çözülür; collection list yüzeyi açılmaz | `ADR-0224`, `0338bf24`, `c57ce5d7` |
| Bounded public relation plan | Tamamlandı (yerel) | Relation resolver/eager-loader için cardinality, depth, edge ve item bütçelerini provider-neutral contract ile sınırlar; authorization/tenant policy Edition'a aittir | `ADR-0230` |
| DX iteration benchmark refresh | Tamamlandı (yerel ölçüm) | Üç koşulu temiz Windows akışında project scaffold/resource generation/compatibility acceptance medianı 1.706 s; Payload eşdeğer koşusu yapılmadan parity iddiası yok | `docs/validation/dx-iteration-2026-09-22-deep-query.json` |
| İki resource golden-path baseline | Tamamlandı (HydroGo + Payload smoke) | Testimonial ve bağımsız Article fixture'ında üçer koşu; median 1.651 s / 1.546 s. Payload 3.90.1 SQLite harness'ında build, cold/warm admin ve warm API smoke ölçüldü; authenticated CRUD, p95/p99 ve kapasite parity'si açık | `docs/validation/R102-hydrogo-golden-path-baseline-2026-09-22.md` |
| Full-suite/static gate refresh | Tamamlandı (yerel Windows) | Full `go test`, `go vet`, HTML comment lint ve ADR drift kapıları yeniden geçti | `docs/validation/R103-full-suite-2026-09-22.md` |
| Generated global singleton DB invariant | Tamamlandı (contract seviyesinde) | `type: global` generated migration'ları `CHECK (id = 'global')` ile ikinci satırı reddeder; SQLite/PostgreSQL operator migration'ı gerekir | `ADR-0238`, generator global contract test |
| Production gate runner PowerShell compatibility | Tamamlandı (yerel script) | Windows PowerShell 5 parser hatası giderildi; URL yoksa fail-closed preflight korunuyor | `docs/validation/R104-production-gate-script-2026-09-22.md` |
| WSL PostgreSQL/Redis production-gate refresh | Tamamlandı (yerel servisler) | Full/shuffled/race, reproducible build, portable benchmark ve Redis atomic limiter kapıları geçti; gerçek restore/proxy/key/capacity tatbikatları açık | `docs/validation/R105-wsl-production-gates-2026-09-22.md` |
| WSL restore/Redis disposable drills | Tamamlandı (yerel servisler) | Disposable PostgreSQL backup/restore + contract ve geçici Redis atomic/race tatbikatları geçti; production proxy/key/capacity dış kapıları açık | `docs/validation/R106-wsl-restore-redis-drills-2026-09-22.md` |
| Generated global Admin surface | Tamamlandı (contract seviyesinde) | Global kaynak list template'i tablo/pagination üretmez; singleton form yüzeyine yönlendirir ve generated test ile korunur | `ADR-0239`, generator global admin test |
| Capacity profile baseline refresh | Tamamlandı (yerel smoke) | Compact/standard/distributed profilleri 100 warm-up + 20 örnekle ölçüldü; peak working set 92.2/94.8/101.5 MiB; müşteri SLA'sı değildir | `docs/validation/R107-capacity-profile-baseline-2026-09-22.md` |
| Global contract static refresh | Tamamlandı (yerel) | Generator global test, ADR drift (261) ve HTML comment lint tekrar geçti | `docs/validation/R108-static-contract-refresh-2026-09-22.md` |
| Post-push security revalidation | Tamamlandı (source-level) | Son pushed global/gate değişiklikleri önceki audit ile yeniden karşılaştırıldı; yeni source-level bulgu yok; provider/penetration kapıları açık | `docs/validation/R109-security-post-push-revalidation-2026-09-22.md` |
| Core over-engineering audit | Tamamlandı (audit) | Whole-repo ponytail incelemesi güvenli silme/dependency replacement bulmadı; deprecated v0 Module removal'ı compatibility window'a ertelendi | `docs/validation/R110-ponytail-audit-2026-09-22.md` |
| Core footprint / P0 baseline | Kısmi (audit; P0 açık) | Windows amd64 universal binary 36.98 MiB; örnek Edition'lar 15.05 MiB/89 non-stdlib package bütçesiyle CI'da ölçülüyor; `profile explain/check` gerçek Edition composition/link graph'ına bağlı değil ve full Edition/inactive resource gate'leri yok | `ADR-0247`, `R120-core-footprint-baseline-2026-09-22.md`, `R121-profile-cli-contract-2026-09-22.md`, `R122-edition-footprint-gate-2026-09-22.md` |
| Bounded Admin saved-view contract | Tamamlandı (yerel) | Kullanıcı görünümleri için kolon/filtre/sort metadata'sını field/operator allowlist ve boyut bütçeleriyle doğrular; execution policy Edition'a aittir | `ADR-0231` |
| Bounded offline sync contract | Tamamlandı (yerel) | Saha/offline Edition'lar için cursor, idempotent command, payload ve batch bütçelerini doğrular; conflict/attachment policy Edition'a aittir | `ADR-0232` |
| Public tenant scope contract | Tamamlandı (yerel) | External Edition boundary için fail-closed tenant context/cache key sözleşmesi; authenticated resolver ve DB RLS/predicate wiring Edition'a aittir | `ADR-0233` |
| Generated Admin view metadata | Tamamlandı (yerel) | Her resource için deterministic `admin/views.json` default view üretir ve compatibility lock ile doğrular; user persistence Edition'a aittir | `ADR-0234` |
| Versioned field validation binding | Tamamlandı (yerel) | Blueprint validation metadata'sı generated `validation.go` içine taşınır ve service Create/Update tarafından çağrılır | `ADR-0226`, `110833a0`, `b985bf7b` |
| Client Edition lifecycle hook scaffold | Tamamlandı (yerel) | Scaffold mevcut custom hook dosyasını ezmeden Before/After create/update/delete iskeleti üretir; yan etkiler outbox'a yönlendirilir | `ADR-0227`, `5a98918d` |
| Public bounded resource query contract | Tamamlandı (yerel) | Client Edition'ların import edebileceği `extension/v1` parser alan/operator allowlist ve query budget'lerini uygular | `ADR-0228`, `c67bdf71` |
| Generated bounded collection list transport | Tamamlandı (yerel) | Generated repository/service/HTTP transport limit-page-offset ve stable sort allowlist ile collection GET sağlar | `ADR-0229`, `00af1350`, `c3123829` |
| Generated route composition helper | Tamamlandı (yerel) | Her resource `runtime.go` içinde policy-neutral `Register<Name>Route` üretir; tenant-scoped kaynaklar resolver ister, migration/worker/middleware Edition'da kalır | `ADR-0240`, generator contract test |
| Versioned Blueprint input boundary | Tamamlandı (yerel) | Blueprint YAML/JSON için explicit/implicit `v0` kabul edilir; desteklenmeyen sürüm artifact üretmeden reddedilir ve JSON Schema ile eşlenir | `ADR-0241`, blueprint loader tests |
| Resource auth composition flag | Tamamlandı (yerel) | Blueprint `auth: true` Spec/module metadata'sına taşınır ve authenticated Identity/Session/RBAC wiring checklist'i üretir; auth sistemi veya grant otomatik oluşturulmaz | `ADR-0242`, blueprint loader tests |
| Generated auth route guard | Tamamlandı (yerel) | `auth: true` generated route registration Edition checker olmadan fail-closed reddedilir ve anonymous request 401 alır; Identity/Session/RBAC implementasyonu Edition-owned kalır | `ADR-0243`, generator auth guard test |
| Generated bounded workflow contract | Tamamlandı (yerel) | Blueprint workflow grafiği 32 state/transition sınırı ve known-state doğrulamasıyla `workflow.json` üretir; transaction, role policy, schedule ve outbox execution module/Edition'a aittir | `ADR-0244`, blueprint workflow tests |
| Blueprint bounded relation metadata ve many-valued storage | Tamamlandı (yerel; PostgreSQL canlı sertifikasyonu açık) | Çoklu relation alanları yalnız normalized tenant-aware join table'a yazılır; item-bound replacement kendi transaction'ında veya Edition transaction'ına katılarak atomik çalışır; hedef yetkisi, eager loading ve cascade Edition/provider sorumluluğudur | `ADR-0245`, `ADR-0246`, generator ve SQLite Edition testleri |
| Generated deep-query binding | Tamamlandı (yerel) | Public query AST generated transporttan parse edilir, field allowlist ve bound SQL ile equals/contains/in/exists/range filtreleri uygulanır | `ADR-0228`, `ADR-0229`, `c28d2746` |
| Bounded HTTP idempotency pilot | Kısmi tamamlandı (yerel) | Site form submit için SQLite/PostgreSQL durable replay/conflict sözleşmesi, generated TypeScript mutation helper'ı, Commerce checkout bounded key + request fingerprint conflict koruması ve Stripe payment-intent provider-key forwarding; iyzico/PayTR provider certificationı açık | `R20-form-idempotency`, `R21-generated-idempotency-client`, `R25-commerce-idempotency-key-validation`, `R26-stripe-payment-intent-idempotency`, `R27-commerce-order-fingerprint`, `ADR-0136` |
| Site inactive-module construction guard | Tamamlandı (yerel) | Site service, forms binding, limiter ve HTTP handler yalnızca Site modülü etkinse oluşturulur; preset route/migration acceptance hâlâ ayrı kanıttır | `R22-site-module-zero-construction`, `R17-wsl-preset-module-isolation` |
| Pages cache inactive-module guard | Tamamlandı (yerel) | Pages kapalı profillerde in-process/Redis page cache oluşturulmaz; metrics provider nil-safe kalır | `R23-pages-cache-zero-construction`, `R17-wsl-preset-module-isolation` |
| Members HTTP inactive-module guard | Tamamlandı (yerel) | Members kapalı profillerde Members HTTP handler/limiter oluşturulmaz; Commerce session çözümlemesi için gereken service korunur | `R24-members-module-zero-construction`, `R17-wsl-preset-module-isolation` |
| Configurable workflow contract | Kısmi tamamlandı (yerel) | Admin Studio’da bounded custom state/transition doğrulaması; persistence ve moderation UI modüle ait | `R5-configurable-workflow` |
| Bounded Admin bulk contract | Kısmi tamamlandı (yerel) | 500 kayıt sınırı, deterministik ID normalizasyonu ve fail-closed istek zarfı; generated `bulk.ts` aynı zarfı üretir; mutation/policy modüle ait | `R5-bounded-admin-bulk-contract`, `R5-generated-bulk-request-adapter` |
| Opt-in Admin bulk list surface | Kısmi tamamlandı (yerel) | `BulkActions` tanımlı collection'larda erişilebilir seçim kutuları ve escaped action metadata; gerçek mutation/CSRF/policy modüle ait | `R5-opt-in-admin-bulk-list-surface` |
| Bounded async relation contract | Kısmi tamamlandı (yerel) | 50 varsayılan/100 hard limit, bounded query/cursor ve deterministic ID/label projection; provider lookup/auth modüle ait | `R5-bounded-async-relation-contract` |
| Locale fallback chain | Kısmi tamamlandı (yerel) | Exact → primary language → supported default sırası, 128 locale sınırı ve fail-closed normalizasyon; public Pages ve News service entegrasyonları eklendi, modül kabul/edition kanıtı açık | `R5-locale-fallback-chain`, `R6-pages-locale-fallback-integration`, `R6-news-locale-fallback-integration` |
| Commerce locale money rendering | Tamamlandı (yerel) | Product, variant, catalog, cart ve checkout yüzeyleri exact integer-cent locale formatter kullanıyor; float-style template kullanımı kaldırıldı | `R28-commerce-locale-money-rendering` |
| Frontend selected design-system metadata | Tamamlandı (yerel) | Generated `frontend/starter.json` artık seçili providerı ve desteklenen provider-agnostic adapter listesini ayrı alanlarda taşır | `R29-frontend-selected-design-system` |
| DX iteration baseline | Kısmi tamamlandı (yerel) | Üç tekrarlı temiz Windows ölçümünde scaffold/resource generation/compatibility acceptance medianı 1.191 saniye; Payload karşılaştırması ve ikinci bağımsız müşteri senaryosu açık | `R30-dx-iteration-benchmark`, `R31-dx-iteration-benchmark-3-runs` |
| Headless starter error/idempotency parity | Tamamlandı (yerel) | Bağımsız TypeScript starter checkout key doğrulaması, typed API error/request-id ve header + streaming bounded response korumasını generated client ile hizalar | `R32-headless-starter-error-idempotency` |
| WSL production-gate preflight (current commit) | Tamamlandı (yerel servis) | PostgreSQL 18.6 + Redis/Valkey üzerinde full/shuffled/race, reproducible build ve portable benchmark yeniden geçti; gerçek restore/proxy/key/capacity kapıları açık | `R33-wsl-production-gates-current` |
| Headless starter client contract smoke | Tamamlandı (yerel) | Derlenmiş TypeScript client için idempotency, typed error/request-id, normal response ve chunked 2 MiB sınırı Node smoke testiyle doğrulanıyor | `R34-headless-starter-client-contract-smoke` |
| Headless starter E2E wiring | Tamamlandı (yerel/CI script) | `scripts/run-e2e.ps1` generated browser journeys öncesinde `examples/headless-starter` contract smoke testini zorunlu çalıştırıyor | `R35-headless-starter-e2e-wiring` |
| Headless starter client bounds | Tamamlandı (yerel) | Contract smoke timeout yapılandırmasının 1–120000 ms fail-closed sınırlarını da doğruluyor | `R36-headless-starter-client-bounds` |
| Headless starter timeout abort | Tamamlandı (yerel) | Contract smoke tamamlanmayan istekte gerçek `AbortController` timeout sinyalini doğruluyor | `R37-headless-starter-timeout` |
| Headless starter local verification gate | Tamamlandı (yerel/CI) | `scripts/verify-local.ps1` npm mevcutsa headless starter smoke testini push doğrulamasına bağlıyor; Node olmayan ortamlarda açık uyarıyla kontrollü skip ediyor | `R38-local-verification-headless-starter` |
| Local Compose loopback bindings | Tamamlandı (yerel) | PostgreSQL, Redis ve MinIO geliştirme portları yalnızca loopback'e bind ediliyor; LAN'a yanlışlıkla servis açılmıyor | `R39-local-compose-loopback-bindings` |
| Compose binding security lint | Tamamlandı (yerel/CI) | Loopback-only Compose binding'leri `verify-local.ps1` içinde fail-closed lint ile korunuyor | `R40-compose-security-lint` |
| Headless starter external abort | Tamamlandı (yerel) | Contract smoke host `AbortSignal` iptalinin internal request controller'a aktarıldığını doğruluyor | `R41-headless-starter-external-abort` |
| Headless starter origin validation | Tamamlandı (yerel) | Client absolute HTTP(S) origin ve credential/scheme sınırlarını request öncesi fail-closed doğruluyor | `R42-headless-starter-origin-validation` |
| Headless starter per-request abort | Tamamlandı (yerel) | Public client metotları çağrı-başı `HydroGoRequestOptions.signal` ile seçici iptali destekliyor | `R43-headless-starter-per-request-abort` |
| Generated client abort/origin parity | Tamamlandı (yerel) | Project generator `HydroClient` çıktısı HTTP(S) baseUrl doğrulaması ve read çağrı-başı abort seçenekleriyle starter sözleşmesini eşliyor | `R44-generated-client-abort-origin-parity` |
| Local Compose image pinning | Tamamlandı (yerel) | MinIO/Redis fixture image'ları release tag'lerine sabitlendi; `:latest` kullanımı Compose lint ile reddediliyor | `R45-compose-image-pinning` |
| Generated client stream cleanup | Tamamlandı (yerel) | Generated bounded response reader `finally` içinde `releaseLock()` ile stream lock sızıntısını önlüyor | `R46-generated-client-stream-lock` |
| Development Compose exact image pins | Tamamlandı (yerel) | PostgreSQL/Redis geliştirme fixture’ları exact patch/release tag’lerine sabitlendi; mutable major tag’ler lint ile reddediliyor | `R47-compose-development-image-pinning` |
| Generated client request ID parity | Tamamlandı (yerel) | Generated `HydroAPIError.requestId` body/header request ID’sini taşıyor; oversize hata yolu da korelasyonu koruyor | `R48-generated-client-request-id` |
| Generated client timeout parity | Tamamlandı (yerel) | Generated `HydroClient` timeout değerini starter ile aynı 1–120000 ms fail-closed sözleşmesinde doğruluyor | `R49-generated-client-timeout-parity` |
| Generated client relative-path boundary | Tamamlandı (yerel) | Generated `HydroClient` mutlak/protocol-relative mutation path'lerini reddediyor; yapılandırılmış API anahtarının yanlış hosta taşınmasını engelliyor | `R50-generated-client-relative-path` |
| WSL PostgreSQL + Redis gate preflight | Tamamlandı (yerel servis) | PostgreSQL 18.6 ve Redis/Valkey üzerinde full/shuffled/race, reproducible build, portable benchmark ve atomic limiter/TTL kanıtı | `R51-wsl-postgresql-redis-gates` |
| Local production drill refresh | Tamamlandı (yerel) | İzole SQLite readiness, backup/release/portable kontratları ve 20 tekrarlı blue-green cache rollback stability geçti | `R52-local-production-drills` |
| Generated client origin-bypass hardening | Tamamlandı (yerel) | URL parser backslash-normalizasyonuyla başka origin'e kaçan request path'leri resolved-origin karşılaştırmasıyla reddediliyor | `R53-generated-client-origin-bypass` |
| S3 object-key/public-base boundary hardening | Tamamlandı (yerel) | Tüm blob işlemleri ortak bounded key doğrulaması kullanıyor; public base credential/query/fragment taşıyamıyor | `R54-s3-object-key-boundary` |
| S3 endpoint URL boundary hardening | Tamamlandı (yerel) | URL biçimli endpoint'lerde HTTP(S), credentialsız ve query/fragmentsız yapılandırma zorunlu | `R55-s3-endpoint-boundary` |
| S3/CDN runtime configuration wiring | Tamamlandı (yerel) | `cdn_base_url` ve `s3_force_path_style` config değerleri çalışan S3 blob store'a aktarılıyor; credential/query/fragment sınırları korunuyor | `R56-s3-config-wiring` |
| Public media CDN resolution | Tamamlandı (yerel) | Public Pages, News ve Commerce server-rendered görselleri optional provider URL’sini kullanıyor; provider yoksa güvenli same-origin fallback korunuyor | `R58-public-media-cdn-consumers` |
| S3/CDN public URL boundary | Kısmi tamamlandı (yerel) | S3-compatible path-style seçeneği, explicit HTTPS public base ve bounded object URL üretimi; private media için 1s–24h presigned URL capability ve protected `/api/media/{id}/url` endpoint’i eklendi; gerçek provider/CDN ve invalidation kanıtı açık | `R6-s3-cdn-public-url-boundary`, `R6-bounded-private-media-urls`, `R6-protected-media-url-endpoint` |
| WSL yerel doğrulama | Tamamlandı (yerel) | Ubuntu/WSL2 üzerinde `go test ./... -p 1 -timeout=5m` | `wsl-full-suite-2026-09-16-r5` |
| WSL statik kalite doğrulaması | Tamamlandı (yerel) | Ubuntu/WSL2 üzerinde `go vet ./...` | `wsl-vet-and-full-suite-2026-09-16` |
| WSL race doğrulaması | Tamamlandı (yerel) | Broker/runtime/release hedef paketleri `go test -race` | `wsl-race-broker-runtime-2026-09-16` |
| OCI container image definition | Tamamlandı (statik yerel) | Non-root, credential-free amd64 Dockerfile, bounded build context ve daemon'sız lint kapısı | `R6-container-image-definition-2026-09-16` |
| Gerçek Redis rate-limit atomic/TTL kanıtı | Tamamlandı (yerel servis) | `HYDRO_TEST_REDIS_URL` ile ilk istek/limit reddi/TTL doğrulaması; servis yoksa test skip | `R6-redis-rate-limit-2026-09-16` |
| PostgreSQL production-gate otomasyonu | Tamamlandı (WSL servis) | PostgreSQL 18.6 üzerinde tam/shuffled/race suite, reproducible build ve portable benchmark | `R6-postgres-production-gates-2026-09-16` |

## Henüz production servisi olmayan parçalar

Bu repodaki SDK katmanı güvenli sözleşmeleri ve yerel artifact işlemlerini sağlar.
Aşağıdaki dış bağımlılıklar gerçek endpoint, hesap ve release altyapısı olmadan
tamamlanmış sayılamaz:

1. OAuth/device-flow login ve token yenileme servisi.
2. GitHub App veya private registry arkasında artifact broker.
3. Cross-platform imzalı release pipeline ve public-key rotation.
4. MSI, Homebrew, deb/rpm ve macOS notarization yayınları.
5. Offline lisans, revocation, mirror ve rollback tatbikatları.
6. S3/CDN sağlayıcısı, backup/restore ve disaster-recovery kanıtı.

Broker endpoint/grant/hata sözleşmesi [HYDRO_SDK_BROKER_API.md](HYDRO_SDK_BROKER_API.md)
belgesinde uygulanabilir biçimde tanımlıdır; gerçek servis değildir.

Private GitHub deposuna kullanıcı PAT'i koymak bu maddelerin yerine geçmez.
İstemci yalnızca kısa ömürlü grant/URL almalı; kaynak kodu değil, imzalı artifact
indirmelidir. Ayrıntılı kararlar için
[`HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md`](HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md)
ve ADR-0091'e bakın.

## Faz bazlı ilerleme

- **Faz 0 (benchmark):** HydroGo iki bağımsız resource fixture'ında ölçüldü
  (median 1.651 s / 1.546 s, yerel Windows); Payload 3.90.1 SQLite harness'ında
  build, cold/warm admin ve warm API smoke ölçüldü. Authenticated CRUD,
  p95/p99 yük testi ve kapasite parity'si açık (`R102`).
- **Faz 1 (resource contract/CLI):** Kısmi. Proje namespace'i, lock yükleme ve
  check akışı var; generated collection'lar artık deterministic `spec.json` ile
  kendini tanımlıyor ve `hydro resource check --root` tüm scaffold'u keşfedip
  doğruluyor; global resource, field validation, lifecycle scaffold ve bounded
  collection list transport eklendi; rich query AST'nin generated transport'a
  tam field/operator binding'i ve migration/admin/API runtime wiring'i hâlâ açık.
- **Faz 2–3 (Admin/editorial):** Açık. Async relation, bulk işlemler, preview
  bridge, schedule ve locale fallback için ürün kabul testleri gerekiyor.
- **Tenant-scoped resource generation:** Tamamlandı (contract seviyesinde).
  Blueprint `tenant_scoped` işareti generated model/repository/service/transport
  yüzeylerini açık tenant parametresi ve fail-closed resolver ile üretir;
  gerçek PostgreSQL RLS, provisioning, quota ve restore kabulü hâlâ üretim
  kapısıdır (ADR-0235).
- **Admin bulk execution primitive:** Tamamlandı (Core contract seviyesinde).
  Module-owned callback ile sıralı, bounded ve callback hata metnini sızdırmayan
  sonuç üretimi eklendi; authorization, idempotency, audit ve transaction
  semantiği Edition/module sorumluluğunda kalıyor (ADR-0236).
- **Edition-extensible tenant RLS verification:** Tamamlandı (contract
  seviyesinde). Provisioning verifier artık generated tenant tablolarını
  fail-closed şekilde doğrulayabiliyor; policy/migration oluşturma ve gerçek
  PostgreSQL restore/capacity kanıtı hâlâ operasyon kapısıdır (ADR-0237).
- **Faz 4 (kit ekosistemi):** Kısmi. Kit/edition sözleşmeleri mevcut; registry,
  dependency çözümleme ve uyumluluk matrisi tamamlanmadı.
- **Faz 5 (frontend SDK/starter):** Kısmi. Adapter sözleşmeleri, typed client/query akışı, disposable SQLite demo ve 47 testli yerel Chromium E2E kanıtı var; harici Client Edition ve production kimlik doğrulama kabulü tamamlanmadı.
- **Headless client hardening:** Tamamlandı (yerel). Starter ve generated istemciler bounded timeout/hata gövdesi sınırı, typed `HydroAPIError` ve request-id taşıyan hata gövdesi kullanır; gerçek browser/kimlik doğrulama kabulü ayrı kapıdır.
- **Faz 6 (production/upgrade):** Açık. Yerel production drill’leri (SQLite
  readiness, backup/release/portable ve blue-green cache rollback) geçti;
  PostgreSQL kapasitesi, gerçek proxy/restore ve imzalı cross-OS release
  kanıtları bekliyor.

## Yerel doğrulama akışı

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

Windows'ta:

```powershell
go test ./internal/sdkdist
go test ./internal/projectgen ./internal/release ./cmd/hydro
```

WSL Ubuntu'da (`whoami` sonucu `operator` olmalı):

```bash
cd /path/to/project
go test ./... -p 1 -timeout=5m
```

Test sonucu ve gözlem süresi `docs/validation/` altında tarihli rapor olarak
eklenmelidir. Şifre veya token raporlara yazılmaz.

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

## Sonraki güvenli sıra

1. Payload benchmark'ını authenticated browser CRUD, seed veri ve p95/p99
   yük testiyle tamamla.
2. Resource contract'ın migration/admin/API runtime wiring kapsamını ölçüm
   sonuçlarına göre genişlet.
3. Login/broker için gerçek servis kararlarını ADR ile kabul et.
4. İmzalı release pipeline ve installer'ları gerçek hesaplarla doğrula.
5. Production kapasite/restore/upgrade tatbikatlarını çalıştır.
