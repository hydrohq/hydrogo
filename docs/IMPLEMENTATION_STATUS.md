# HydroGo uygulama durumu ve çalışma rehberi

Bu belge, planlanan işlerle gerçekten teslim edilmiş işleri ayırır. Bir özelliğin
burada “tamamlandı” olması, kodun repoda bulunması ve yerel testinin geçmesi
anlamına gelir; production hazır olduğu anlamına gelmez.

## Tamamlanan ve doğrulanan parçalar

| Alan | Durum | Nasıl kullanılır? | Kanıt |
| --- | --- | --- | --- |
| Public proje komut alanı | Tamamlandı | `hydro project new`, `hydro project check` | `R5-sdk-project-namespace`, `R5-sdk-project-check` |
| Üretilen SDK rehberi | Tamamlandı | Yeni projede `docs/SDK.md` | `R5-generated-sdk-guide` |
| Runtime artifact doğrulama | Tamamlandı | `hydro runtime verify [--root PATH]` | `R5-runtime-artifact-verification` |
| Platform hedefi ve artifact çözümleme | Tamamlandı | `linux-amd64`, `windows-arm64` vb. hedefler | `R5-sdk-platform-targets`, `R5-sdk-artifact-resolution` |
| Güvenli runtime cache yolu | Tamamlandı | Ürün/sürüm/hedef bazlı deterministik cache | `R5-sdk-runtime-cache` |
| Atomik artifact kurulumu | Tamamlandı | checksum/size doğrulaması + atomic rename | `R5-sdk-atomic-install` |
| Download URL ve bounded download | Tamamlandı | HTTPS, redirect kapalı, boyut sınırı | `R5-sdk-download-url-policy`, `R5-sdk-bounded-download` |
| Grant doğrulama ve indirme bağı | Tamamlandı | proje/core/sdk/expiry/url kontrolleri | `R5-sdk-artifact-grant`, `R5-sdk-grant-download-integration`, `R5-sdk-grant-parser` |
| WSL yerel doğrulama | Devam ediyor | Ubuntu/WSL2 üzerinde `go test ./...` | `wsl-local-baseline` |

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

Private GitHub deposuna kullanıcı PAT'i koymak bu maddelerin yerine geçmez.
İstemci yalnızca kısa ömürlü grant/URL almalı; kaynak kodu değil, imzalı artifact
indirmelidir. Ayrıntılı kararlar için
[`HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md`](HYDRO_SDK_PRIVATE_RUNTIME_DISTRIBUTION.md)
ve ADR-0091'e bakın.

## Faz bazlı ilerleme

- **Faz 0 (benchmark):** Açık. Payload eşdeğer haber örneği ve iki müşteri
  akışıyla süre/manuel adım ölçümü yapılmadı.
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

1. WSL tam testini tamamla ve kanıt raporunu güncelle.
2. Resource contract kapsamını ve benchmark senaryosunu netleştir.
3. Login/broker için gerçek servis kararlarını ADR ile kabul et.
4. İmzalı release pipeline ve installer'ları gerçek hesaplarla doğrula.
5. Production kapasite/restore/upgrade tatbikatlarını çalıştır.
