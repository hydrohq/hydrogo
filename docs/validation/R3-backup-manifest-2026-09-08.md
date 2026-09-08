# R3 Compact SQLite snapshot doğrulaması

**Tarih:** 2026-09-08  
**Durum:** Snapshot/manifest, byte/süre bütçesi ve SQLite restore control state'i başarılı; temiz ortam restore ve PostgreSQL kapısı açık.

`internal/backup` içindeki `SnapshotSQLite` kontrollü yazma durdurması varsayımıyla `VACUUM INTO` kullanarak staging dizinine SQLite snapshot'ı alır. Medya dosyaları `media/` altında kopyalanır ve SHA-256/byte boyutu manifest'e yazılabilir. `Verify` aktivasyon yapmadan veritabanı ve medya bütünlüğünü doğrular; medya değişikliği ve traversal/duplicate medya manifest path'leri testte reddedilir. `RestoreToStaging` yalnız boş hedefe, doğrulama sonrasında kopyalar ve session iptali, dış egress karantinası ve eski yazmaların durdurulduğunu belirten marker olmadan başarılı olmaz.

`202609090015_restore_controls` migration'ı restore state'ini DB'ye taşır. `202609090016_restore_write_fence` kritik pages/media/news/jobs tablolarına DB trigger'ı ekler. `ApplyRestoreControls` tüm session'ları iptal eder ve `restore_controls` satırını etkinleştirir; job store quarantine aktifken claim döndürmez, eski deployment yazıları trigger tarafından reddedilir. Staging doğrulamasından sonra `ReleaseRestoreQuarantine` job claim'lerini açar; eski slot drain edildikten sonra `ReleaseRestoreWrites` write fence'ini kaldırır. SQLite ve izole PostgreSQL schema contract'ları aynı kontrol sırasını doğrular.

`SnapshotSQLiteWithLimits` medya toplamı, SQLite page estimate'i/snapshot byte boyutu ve wall-clock bütçesini staging kopyası başlamadan ve kopya sırasında kontrol eder. Bütçe aşılırsa yeni destination temizlenir ve kullanılabilir restore marker'ı üretilmez. Snapshot dizini artık `manifest.json` ile kendi doğrulama girdisini taşır; contract test dönen manifest ile disk üzerindeki manifest'in eşleştiğini kontrol eder. `PostgresTools` ise password'ı process argümanına koymadan geçici `PGPASSFILE` ile custom-format `pg_dump`/`pg_restore` çağrısını sınırlar; `SnapshotPostgres` ve `RestorePostgresToStaging` manifest doğrulamalı medya kopyası, karantina marker'ı ve izole hedef database round-trip'ini CI client araçlarıyla doğrular. `ActivateStagingMedia` üç kontrolü taşıyan marker olmadan çalışmaz, staging kökünü aktif köke atomik olarak geçirir ve eski kökü önceden boş rollback dizininde korur.

`PruneSnapshots`, yalnız geçerli `manifest.json` taşıyan yerel snapshot dizinlerini değerlendirir; en yeni snapshot sayısını ve pin'li dizinleri korur, bozuk/yarım manifest'te güvenli biçimde durur. Bu yardımcı dış object storage GC veya erasure log retention koordinasyonunun yerine geçmez.

Çalıştırılan kontrol:

```text
go test ./internal/backup
go test ./...
```

Bu deney uygulama CLI'sına restore komutu eklemez, medya aktivasyonunu, PostgreSQL restore'ı sonrasında gerçek trafik geçişini veya gerçek eski binary yazma reddini kanıtlamaz. Bunlar S07'nin sonraki adımlarıdır.
