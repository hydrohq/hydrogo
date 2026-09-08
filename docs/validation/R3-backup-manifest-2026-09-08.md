# R3 Compact SQLite snapshot doğrulaması

**Tarih:** 2026-09-08  
**Durum:** Snapshot/manifest deneyi başarılı; temiz ortam restore ve PostgreSQL kapısı açık.

`internal/backup` içindeki `SnapshotSQLite` kontrollü yazma durdurması varsayımıyla `VACUUM INTO` kullanarak staging dizinine SQLite snapshot'ı alır. Medya dosyaları `media/` altında kopyalanır ve SHA-256/byte boyutu manifest'e yazılabilir. `Verify` aktivasyon yapmadan veritabanı ve medya bütünlüğünü doğrular; medya değişikliği testte reddedilir.

Çalıştırılan kontrol:

```text
go test ./internal/backup
go test ./...
```

Bu deney uygulama CLI'sına restore komutu eklemez, PostgreSQL restore'ını veya session/job/egress karantinasını kanıtlamaz. Bunlar S07'nin sonraki adımlarıdır.
