# R5 taşınabilir export/import doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R5 / S09  
**Durum:** Allowlist tabanlı SQLite round-trip başarılı; PostgreSQL round-trip ve büyük veri benchmark'ı açık.

`internal/portable` deterministic JSON formatında pages, page revisions, media metadata, Site Edition, form inbox ve News kayıtlarını taşır. Kimlik, revision, locale ve ilişkiler korunur; `users`, `sessions`, MFA/recovery/invite token'ları, audit, outbox ve jobs tabloları formatın allowlist'ine dahil değildir.

Çalıştırılan kontrol:

```text
go test ./internal/portable
```

Contract test source SQLite'dan export alıp ayrı SQLite hedefe import eder; page ve revision korunur, auth state'in dışarı çıkmadığı doğrulanır. PostgreSQL dialect round-trip ve secret dışa aktarma denetimi CI/temiz ortam kapısında ayrıca çalıştırılmalıdır.
