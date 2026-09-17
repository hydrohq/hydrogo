# HydroGo uygulama durumu ve çalışma rehberi

Bu belge, planlanan işlerle gerçekten teslim edilmiş işleri ayırır. Bir özelliğin
burada “tamamlandı” olması, kodun repoda bulunması ve yerel testinin geçmesi
anlamına gelir; production hazır olduğu anlamına gelmez.

## Tamamlanan ve doğrulanan parçalar

| Alan | Durum | Nasıl kullanılır? | Kanıt |
| --- | --- | --- | --- |
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

- **Faz 0 (benchmark):** HydroGo baseline ölçüldü (23.454 s, yerel Windows);
  Payload eşdeğer haber örneği ve iki müşteri akışıyla göreli ölçüm açık.
- **Faz 1 (resource contract/CLI):** Kısmi. Proje namespace'i, lock yükleme ve
  check akışı var; generated collection'lar artık deterministic `spec.json` ile
  kendini tanımlıyor ve `hydro resource check --root` tüm scaffold'u keşfedip
  doğruluyor; migration/admin/API runtime wiring'i hâlâ açık.
- **Faz 2–3 (Admin/editorial):** Açık. Async relation, bulk işlemler, preview
  bridge, schedule ve locale fallback için ürün kabul testleri gerekiyor.
- **Faz 4 (kit ekosistemi):** Kısmi. Kit/edition sözleşmeleri mevcut; registry,
  dependency çözümleme ve uyumluluk matrisi tamamlanmadı.
- **Faz 5 (frontend SDK/starter):** Kısmi. Adapter sözleşmeleri, typed client/query akışı, disposable SQLite demo ve 47 testli yerel Chromium E2E kanıtı var; harici Client Edition ve production kimlik doğrulama kabulü tamamlanmadı.
- **Headless client hardening:** Tamamlandı (yerel). Starter ve generated istemciler bounded timeout/hata gövdesi sınırı, typed `HydroAPIError` ve request-id taşıyan hata gövdesi kullanır; gerçek browser/kimlik doğrulama kabulü ayrı kapıdır.
- **Faz 6 (production/upgrade):** Açık. Yerel production drill’leri (SQLite
  readiness, backup/release/portable ve blue-green cache rollback) geçti;
  PostgreSQL kapasitesi, gerçek proxy/restore ve imzalı cross-OS release
  kanıtları bekliyor.

## Yerel doğrulama akışı

Windows'ta:

```powershell
go test ./internal/sdkdist
go test ./internal/projectgen ./internal/release ./cmd/hydro
```

WSL Ubuntu'da (`whoami` sonucu `arda` olmalı):

```bash
cd /mnt/c/dev/cms
go test ./... -p 1 -timeout=5m
```

Test sonucu ve gözlem süresi `docs/validation/` altında tarihli rapor olarak
eklenmelidir. Şifre veya token raporlara yazılmaz.

## Sonraki güvenli sıra

1. Payload benchmark protokolünü iki bağımsız resource senaryosunda çalıştır.
2. Resource contract'ın otomatik wiring kapsamını ölçüm sonuçlarına göre genişlet.
3. Login/broker için gerçek servis kararlarını ADR ile kabul et.
4. İmzalı release pipeline ve installer'ları gerçek hesaplarla doğrula.
5. Production kapasite/restore/upgrade tatbikatlarını çalıştır.
