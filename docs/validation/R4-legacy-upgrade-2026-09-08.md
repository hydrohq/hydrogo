# R4 legacy müşteri upgrade doğrulaması

**Tarih:** 2026-09-08  
**Durum:** Başarılı

`tests/contract/testdata/legacy_customer.json`, migration `202609080008` seviyesinde oluşturulmuş gerçekçi bir müşteri snapshot'ını temsil eder. Contract test önce bu eski şemayı kurup sayfa, revision, hazır medya ve kullanıcı hash'ini yazar; ardından tüm migration setini uygular ve `CheckCompatibility` ile checksum sözleşmesini doğrular.

Upgrade sırasında `202609080014_published_projection_backfill` migration'ı, projection kolonları eklenmeden önce yayınlanmış olan sayfaları yeni public projection'a taşır. Test, eski sayfanın public route üzerinden okunabildiğini, medya storage key/status değerlerinin ve parola hash'inin değişmediğini kanıtlar.

Çalıştırılan kontroller:

```text
go test ./tests/contract -run Legacy -count=1
go test ./...
go vet ./...
```

Bu fixture SQLite üzerinde çalışır; PostgreSQL migration ve portable round-trip sözleşmeleri ayrı contract testlerinde çalıştırılır.
