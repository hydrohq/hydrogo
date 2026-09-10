# Destek matrisi başlangıç kanıtı

Bu tablo satılabilir destek garantisi değildir; her satırın yanındaki CI veya contract kanıtı tamamlanmadan profil desteklenmiş ilan edilmez.

| Alan | CI/contract kanıtı | Durum |
|---|---|---|
| Go toolchain | Go 1.27.0 ile Linux ve Windows build/test/vet | Doğrulanıyor |
| SQLite | `modernc.org/sqlite` v1.58.0; migration, rollback, export/import contract'ları | Teknik contract hazır |
| PostgreSQL | PostgreSQL 18 service; migration, transaction, restore ve export/import contract'ları | Teknik contract hazır |
| PostgreSQL backup araçları | PostgreSQL 18 `pg_dump`/`pg_restore`; custom-format dump/restore round-trip | Teknik contract hazır |
| Linux amd64 | `ubuntu-latest` CI build/test ve reproducible binary | CI doğruluyor |
| Windows amd64 | `windows-latest` CI build/test ve reproducible binary | CI doğruluyor |
| Compact profil | Typed budget ve `hydro doctor`; gerçek müşteri benchmark'ı | Açık |
| Standard profil | Typed budget; gerçek PostgreSQL pool/throughput benchmark'ı | Açık |
| Distributed profil | Mimari sözleşme mevcut; çoklu instance/L2/S3/CDN tatbikatı yok | Açık |
| Modüller (Pages & News) | SQLite & PostgreSQL contract testleri, admin/API akışları | Teknik contract hazır (Pilot adayı) |
| Modül (Commerce) | Domain/SQL store/checkout contract testleri, webhook & sandbox provider testleri | Deneysel / Pilot adayı (Prod sertifikasyonu açık) |

Sürüm yükseltme, provider ve işletim sistemi desteği; gerçek benchmark, temiz ortam restore ve release/deprecation takvimi tamamlandıktan sonra ayrı bir release kararıyla sabitlenir.
