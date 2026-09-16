# HydroGo uygulama durumu ve çalışma rehberi

Bu belge, planlanan işlerle gerçekten teslim edilmiş işleri ayırır. Bir özelliğin
burada “tamamlandı” olması, kodun repoda bulunması ve yerel testinin geçmesi
anlamına gelir; production hazır olduğu anlamına gelmez.

## Tamamlanan ve doğrulanan parçalar

| Alan | Durum | Nasıl kullanılır? | Kanıt |
| --- | --- | --- | --- |
| Public proje komut alanı | Tamamlandı | `hydro project new`, `hydro project check` | `R5-sdk-project-namespace`, `R5-sdk-project-check` |
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
| Typed frontend query adapter | Kısmi tamamlandı (yerel) | Üretilen `react.ts` içinde framework-neutral typed query options ve stabil query key yardımcıları | `R5-typed-frontend-query-adapter` |
| Taxonomy contract | Kısmi tamamlandı (yerel) | Çok dilli kategori/etiket, parent cycle ve related-content referanslarını bounded doğrulayan provider-neutral sözleşme | `R5-taxonomy-contract` |
| Search contract | Kısmi tamamlandı (yerel) | Bounded query/locale/pagination ve deterministik sonuç sözleşmesi; provider/index seçimi açık | `R5-search-contract` |
| Generated SWR adapter | Kısmi tamamlandı (yerel) | Kaynaksız dependency-free typed key/fetcher çıktısı; browser entegrasyonu açık | `R5-generated-swr-adapter` |
| Form contract | Kısmi tamamlandı (yerel) | Bounded metadata ve submission normalizasyonu; inbox/consent/delivery modüle ait | `R5-form-contract` |
| Kit dependency resolution | Kısmi tamamlandı (yerel) | Allowlisted dependency expansion ve deterministic kit set; generated admin/SDK/runtime artifacts now include resolved dependencies; implementation loading modüle ait | `R5-kit-dependency-resolution`, `R5-generated-kit-dependency-artifacts` |
| Generated form schema | Kısmi tamamlandı (yerel) | Collection spec’inden deterministic `form.schema.json`; business validation/inbox modüle ait | `R5-generated-form-schema` |
| Frontend starter manifest | Kısmi tamamlandı (yerel) | Preset/presentation/features/design-system seçeneklerini bildiren credential-free manifest | `R5-frontend-starter-manifest` |
| Configurable workflow contract | Kısmi tamamlandı (yerel) | Admin Studio’da bounded custom state/transition doğrulaması; persistence ve moderation UI modüle ait | `R5-configurable-workflow` |
| WSL yerel doğrulama | Tamamlandı (yerel) | Ubuntu/WSL2 üzerinde `go test ./... -p 1 -timeout=5m` | `wsl-full-suite-2026-09-16-r5` |
| WSL statik kalite doğrulaması | Tamamlandı (yerel) | Ubuntu/WSL2 üzerinde `go vet ./...` | `wsl-vet-and-full-suite-2026-09-16` |
| WSL race doğrulaması | Tamamlandı (yerel) | Broker/runtime/release hedef paketleri `go test -race` | `wsl-race-broker-runtime-2026-09-16` |

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
  check akışı var; tek kaynaklı resource'tan tüm migration/admin/API wiring'i
  henüz otomatik üretilmiyor.
- **Faz 2–3 (Admin/editorial):** Açık. Async relation, bulk işlemler, preview
  bridge, schedule ve locale fallback için ürün kabul testleri gerekiyor.
- **Faz 4 (kit ekosistemi):** Kısmi. Kit/edition sözleşmeleri mevcut; registry,
  dependency çözümleme ve uyumluluk matrisi tamamlanmadı.
- **Faz 5 (frontend SDK/starter):** Açık. Adapter sözleşmeleri var; typed
  client/hooks ve starter setleri üretim kabulünden geçmedi.
- **Faz 6 (production/upgrade):** Açık. PostgreSQL kapasitesi, blue/green,
  restore ve imzalı cross-OS release kanıtları bekliyor.

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
