# R3 Site Edition veri sözleşmesi doğrulaması

**Tarih:** 2026-09-08  
**Durum:** Settings/navigation/redirect/form inbox store kapısı başarılı; public settings/navigation projection'ı, canonical/OG helper'ı, published-only sitemap HTTP route'u, admin-only mutation route'ları, allowlist/rate-limited contact submit, inbox status/purge ve transaction içi audit işlemleri hazır.

`202609080009_site_edition` migration'ı her iki dialect için site settings, sıralı navigation, 301/308 redirect ve sınırlı form inbox tablolarını ekler. `modules/site` doğrulama katmanı key/path/URL/e-posta/payload boyutlarını sınırlar; SQL store görünür settings ve navigation'ı yalnız explicit public/visible alanlardan okur. Form submission kayıtları source IP'nin ham değeri yerine hash bekler.

`202609080011_navigation_position_unique` ileri migration'ı locale içindeki navigation position değerini DB seviyesinde de unique yapar; servis doğrulamasını aşan eşzamanlı/direct writes contract testinde reddedilir.

Çalıştırılan kontrol:

```text
go test ./tests/contract ./modules/site/...
go test ./...
```

SQLite contract geçti; PostgreSQL testi `HYDRO_TEST_POSTGRES_URL` yoksa atlanır. Canonical/OG değerleri ve sitemap XML helper'ı unit testlidir; sitemap route'u yalnız `ListPublishedRoutes` projection'ından beslenir. Admin mutation route'ları yalnız aktif admin context'i kabul eder. Contact submit yalnız `contact` allowlist'ini, 64 KiB body sınırını ve bounded IP/form limiter'ını kullanır. Admin inbox status geçişleri allowlistlidir; retention purge ve site mutation audit event'leri contract'te doğrulanır.
