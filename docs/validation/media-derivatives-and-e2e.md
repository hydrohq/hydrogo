# Medya Türevleri ve Otomatik E2E Durum Raporu

Tarih: 2026-09-12  
Durum: Yerel kod ve test doğrulaması tamamlandı; gerçek encoder kurulumu açık kapıdır.

## Medya optimizasyonu

Medya yükleme akışı, orijinal dosyayı immutable kaynak olarak koruyacak şekilde genişletildi. Görsel ve video türevleri ayrı metadata kayıtlarında tutulur:

- `media_assets` orijinal dosyanın kaydıdır ve mevcut sunum davranışı korunur.
- `media_variants` WebP/WebM türevlerinin kimlik, durum, MIME, boyut ve hata bilgisini tutar.
- `?variant=card` ve `?variant=video` istekleri hazır türevi kullanır; türev hazırlanmadıysa orijinale güvenli biçimde düşer.
- JPEG, PNG, WebP yanında MP4, WebM ve QuickTime yüklemeleri kabul edilir.
- Admin medya formu görsel ve video yüklemeyi destekler; video önizlemesi metadata hazır olduğunda gösterilir.

Dönüştürme upload isteğinde çalışmaz. `internal/media.Optimizer` sabit kapasiteli kuyruk ve sınırlı worker ile arka planda çalışır. Her iş için 2 dakika timeout uygulanır; FFmpeg tek codec thread’iyle çağrılır. Kuyruk dolduğunda yükleme ve orijinal medya etkilenmez.

Encoder yolu `HYDRO_MEDIA_FFMPEG` ile verilebilir. Değişken boşsa PATH içindeki `ffmpeg` aranır. Bu makinede FFmpeg bulunmadığı için gerçek WebP/WebM üretimi henüz çalıştırılmamıştır; bu durumda orijinal dosya kullanılabilir kalır ve türev `failed` durumunda bounded hata mesajıyla kaydedilir.

## RAM ve kapasite politikası

- Orijinal upload akışlı ve mevcut `MediaMaxBytes` sınırıyla korunur.
- Türev kuyruğu en fazla 8 bekleyen asset taşır.
- Worker sayısı profil değerinden gelir ve medya için en fazla 2 ile sınırlandırılır.
- Compact profilde normal kullanımda tek dönüştürme çalışır.
- FFmpeg işlerinde tek codec thread’i ve işlem timeout’u kullanılır.
- Orijinal dosya silinmediği için encoder hatası veri kaybına dönüşmez.

Bu sınırlar RAM’in yükleme sayısıyla sınırsız büyümesini engeller. FFmpeg’in işletim sistemi seviyesinde ayrı bir RAM kotası henüz uygulanmamıştır; yüksek hacimli dağıtımlarda container/job-object limiti ve durable media job worker kararı ayrıca alınmalıdır.

## Mimari karar

Karar `docs/architecture/0026-bounded-media-derivatives.md` içinde kayıtlıdır. Yeni migration:

- `migrations/202609120001_media_variants`

RabbitMQ/BullMQ eklenmedi. Mevcut HydroGo sınırları ve Compact felsefesi gereği ilk uygulama bounded in-process worker olarak bırakıldı. Süreç yeniden başlatılırsa bekleyen türev işleri kaybolabilir; orijinal medya etkilenmez. Çok prosesli dayanıklılık gerektiğinde mevcut SQL job queue üzerine durable media job tipi eklenmelidir.

## Otomatik Playwright E2E

Metadata tabanlı E2E generator genişletildi. `e2e/manifest.json` içindeki her frontend route için generator artık şunları üretir:

- Desktop route smoke testi
- 390x844 mobil viewport testi
- Root test ID görünürlük testi
- Yatay taşma (`scrollWidth`) kontrolü

Admin tarafında medya yükleme journey’si de manifest’e eklendi. Test küçük bir PNG yükler, redirect’i ve medya kartındaki dosya adını doğrular. Böylece medya upload yüzeyi üretilebilir E2E kapsamına dahil edilmiştir.

Üretilen dosya:

- `e2e/generated/hydro.spec.ts`

`scripts/run-e2e.ps1`, izole SQLite sunucusu ve test-only yönetici parolasıyla çalıştırıldı:
Chromium üzerinde son koşuda 30/30 test geçti. Koşu; yayınlanmış içerik API istemcisini, yönetici girişini,
beş yönetici alanını, medya yüklemeyi, Pages/News formlarını, public home/shop route'larını ve
iki mobil yatay-taşma kontrolünü kapsar. Aynı koşu artık `.github/workflows/ci.yml` içindeki
zorunlu `browser-e2e` işiyle Windows CI'da da çalışır. Runner, geliştirici makinesinden kalmış
`HYDRO_REDIS_URL` değerini geçici olarak temizleyerek Compact SQLite fixture'ının izolasyonunu korur.

## Test kanıtı

Başarılı kontroller:

- `go test ./tools/e2egen -p 1 -timeout=5m`
- `go test ./migrations ./internal/media/... ./cmd/hydro/... -p 1 -timeout=5m`
- `go vet ./...`
- `git diff --check`
- `scripts/run-e2e.ps1` — Chromium üzerinde 30/30 PASS

## Açık maddeler

- `[!]` Test ortamına FFmpeg kurulması ve gerçek WebP/WebM dönüşüm smoke testinin çalıştırılması.
- `[?]` Çok prosesli üretim için medya işlerinin SQL durable queue’ya taşınması.
- `[?]` FFmpeg için container/job-object seviyesinde sert RAM/CPU kotası.
- `[?]` Her frontend için modül-spesifik cart, haber detay, üye auth, empty/error ve erişilebilirlik E2E akışlarının manifest’e eklenmesi.
