# HydroGo — Güncel Durum

**Son kontrol:** 14 Eylül 2026  
**Dal:** `main`  
**Git durumu:** Temiz, `origin/main` ile senkron  
**Task özeti:** **60 tamamlandı / 20 açık / 80 toplam**

Bu dosya, çalışma sırasında ekranda görünmeyen ilerleme mesajlarının kalıcı
özetidir. Ana ayrıntılı liste `docs/PROFESSIONALIZATION_MASTER_TASKS.md` içinde,
harici kapılar ise `docs/OPEN_GATES_AND_OPERATOR_ACTIONS.md` içindedir.

## Son yapılanlar

- `48fd9fd` — escape edilmiş ortak server-rendered UI primitive’leri eklendi:
  Button, Field, FormError, Table, Pagination, Empty, Loading ve Error.
- `f1576c2` — Commerce katalog, sepet, checkout ve sipariş sayfalarındaki sabit
  `/tr/` public bağlantıları yapılandırılmış locale’e bağlandı.
- `9574885` — izole SQLite üzerinde Chromium Playwright **30/30** E2E geçti;
  kanıt `docs/validation/R6-browser-e2e-2026-09-14.md` dosyasına yazıldı.
- `c3f6656` — Member account sayfasındaki `<html lang>` yapılandırılmış
  locale ile eşleştirildi.
- Full local verification her anlamlı commit’te çalıştı: modül bütünlüğü,
  vulnerability scan, SBOM, asset guard, gofmt, vet, tam/shuffled test,
  fuzz smoke, build, generator smoke ve health/readiness geçti.

## Açık tasklar

### Harici altyapı / production kanıtı

- Harici penetration test ve threat-model incelemesi.
- Gerçek production HTTPS/proxy trust/secure cookie/HSTS/CSP doğrulaması.
- Gerçek Redis/Valkey failure, failover, clock-skew ve multi-instance testi.
- PostgreSQL contract/restore, blue-green rollback ve drain tatbikatları.
- Compact/Standard/Distributed müşteri ölçekli kapasite ölçümleri.
- 24 saatlik soak/spike/media/FFmpeg/backup testleri.
- Gerçek ödeme sağlayıcısı sandbox callback, signature, idempotency, refund ve
  timeout doğrulaması.
- S3/CDN, durable queue ve Redis/Valkey production sınırlarının kararı.

### Frontend / ürün geliştirmeleri

- Desktop/tablet/mobile görsel regression baseline’ları.
- RTL, uzun metin, locale, tarih, sayı ve para birimi fixture matrisi.
- Auth, editor, preview, media picker, cart ve checkout için refresh,
  back-forward ve cross-origin browser senaryoları.
- Taxonomy: kategori, tag, multilingual label ve related content sözleşmesi.
- Configurable editorial workflow ve moderation geçişleri.
- Form builder, validation ve inbox retention policy.
- Gerçek sorgu ölçümünden sonra search provider sözleşmesi.
- Privacy ADR ile durable analytics/consent modeli.

### Bilinçli ürün / ticari kararlar

- OIDC/SAML, mobile API, multi-tenant, AI ve Distributed kapsam kararı.
- SDK v1 support window, LTS, SLA, licensing ve deprecation policy.
- Kullanılmayan demo endpoint/dead route/duplicate helper temizliğinin geniş
  release audit’i.

## Önemli sınır

Harici credentials, gerçek production servisi, müşteri verisi veya ürün/hukuk
kararı gerektiren maddeler yerel mock’larla tamamlanmış sayılmadı. Bu nedenle
listede açık görünmeleri beklenen ve bilinçli bir durumdur.

## Komutlar

```powershell
# Tam lokal doğrulama
powershell -ExecutionPolicy Bypass -File scripts/verify-local.ps1

# Browser E2E (test-only parola gerekir)
$env:HYDRO_E2E_ADMIN_PASSWORD = 'ci-only-e2e-password'
powershell -ExecutionPolicy Bypass -File scripts/run-e2e.ps1

# Task listesini say
$done = (Select-String docs/PROFESSIONALIZATION_MASTER_TASKS.md '- \[x\]').Count
$open = (Select-String docs/PROFESSIONALIZATION_MASTER_TASKS.md '- \[ \]').Count
Write-Output "done=$done open=$open total=$($done+$open)"

# Tek komutla durum özeti
powershell -ExecutionPolicy Bypass -File scripts/status.ps1
```
