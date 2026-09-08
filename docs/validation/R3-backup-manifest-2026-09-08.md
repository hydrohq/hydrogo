# R3 Compact SQLite snapshot doğrulaması

**Tarih:** 2026-09-08  
**Durum:** Snapshot/manifest deneyi ve SQLite restore control state'i başarılı; temiz ortam restore ve PostgreSQL kapısı açık.

`internal/backup` içindeki `SnapshotSQLite` kontrollü yazma durdurması varsayımıyla `VACUUM INTO` kullanarak staging dizinine SQLite snapshot'ı alır. Medya dosyaları `media/` altında kopyalanır ve SHA-256/byte boyutu manifest'e yazılabilir. `Verify` aktivasyon yapmadan veritabanı ve medya bütünlüğünü doğrular; medya değişikliği ve traversal/duplicate medya manifest path'leri testte reddedilir. `RestoreToStaging` yalnız boş hedefe, doğrulama sonrasında kopyalar ve session iptali, dış egress karantinası ve eski yazmaların durdurulduğunu belirten marker olmadan başarılı olmaz.

`202609090015_restore_controls` migration'ı restore state'ini DB'ye taşır. `202609090016_restore_write_fence` kritik pages/media/news/jobs tablolarına DB trigger'ı ekler. `ApplyRestoreControls` tüm session'ları iptal eder ve `restore_controls` satırını etkinleştirir; job store quarantine aktifken claim döndürmez, eski deployment yazıları trigger tarafından reddedilir. Staging doğrulamasından sonra `ReleaseRestoreQuarantine` job claim'lerini açar; eski slot drain edildikten sonra `ReleaseRestoreWrites` write fence'ini kaldırır.

Çalıştırılan kontrol:

```text
go test ./internal/backup
go test ./...
```

Bu deney uygulama CLI'sına restore komutu eklemez, PostgreSQL restore'ını veya gerçek eski binary yazma reddini kanıtlamaz. Bunlar S07'nin sonraki adımlarıdır.
