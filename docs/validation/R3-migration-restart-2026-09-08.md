# R3 migration restart ve uyumluluk doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R3 / S06 başlangıç kapısı  
**Durum:** SQLite restart/checksum kapısı başarılı; online DDL, rollback ve retained asset tatbikatı açık.

Temiz SQLite dosyasında tüm migration'lar uygulanıp `schema_migrations` ledger'ı doğrulandı. DB kapatılıp yeniden açıldıktan sonra ikinci `ApplyMigrations` çağrısı no-op oldu; migration satır sayısı değişmedi. Ledger checksum'ı değiştirilmiş gibi simüle edildiğinde `CheckCompatibility` servise başlamayı reddetti. Backup staging testi retained medya dosyasının doğrulama sonrası aynı byte içeriğiyle restore edildiğini de kanıtlar.

```text
go test ./internal/database
```

Bu test eski binary ile yeni yazma/rollback uyumluluğunu, online DDL kilit davranışını veya medya retained asset sürekliliğini tek başına sertifikalamaz.
