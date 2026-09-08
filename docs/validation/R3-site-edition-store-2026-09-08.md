# R3 Site Edition veri sözleşmesi doğrulaması

**Tarih:** 2026-09-08  
**Durum:** Settings/navigation/redirect/form inbox store kapısı başarılı; public SEO HTTP çıktısı ve admin yetki akışı açık.

`202609080009_site_edition` migration'ı her iki dialect için site settings, sıralı navigation, 301/308 redirect ve sınırlı form inbox tablolarını ekler. `modules/site` doğrulama katmanı key/path/URL/e-posta/payload boyutlarını sınırlar; SQL store görünür settings ve navigation'ı yalnız explicit public/visible alanlardan okur. Form submission kayıtları source IP'nin ham değeri yerine hash bekler.

Çalıştırılan kontrol:

```text
go test ./tests/contract ./modules/site/...
go test ./...
```

SQLite contract geçti; PostgreSQL testi `HYDRO_TEST_POSTGRES_URL` yoksa atlanır. HTTP/admin yüzeyi, canonical/OG/sitemap üretimi ve retention/rate limit politikası sonraki Site Edition kapılarıdır.
