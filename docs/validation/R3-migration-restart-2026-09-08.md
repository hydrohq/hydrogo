# R3 migration restart ve uyumluluk doğrulaması

**Tarih:** 2026-09-08  
**Roadmap:** R3 / S06 başlangıç kapısı  
**Durum:** SQLite restart/checksum ve expand/rollback uyumluluk başlangıç kapıları başarılı; gerçek blue/green trafik ve tam profil ölçümü açık.

Temiz SQLite dosyasında tüm migration'lar uygulanıp `schema_migrations` ledger'ı doğrulandı. DB kapatılıp yeniden açıldıktan sonra ikinci `ApplyMigrations` çağrısı no-op oldu; migration satır sayısı değişmedi. Ledger checksum'ı değiştirilmiş gibi simüle edildiğinde `CheckCompatibility` servise başlamayı reddetti. Backup staging testi retained medya dosyasının doğrulama sonrası aynı byte içeriğiyle restore edildiğini de kanıtlar.

```text
go test ./internal/database
```

Bu test eski binary ile yeni yazma/rollback uyumluluğunu, online DDL kilit davranışını veya gerçek slot trafiğini tek başına sertifikalamaz.

## S06 ilerleme doğrulaması — 2026-09-09

`tests/contract/migration_rollback_test.go` tam migration seti uygulandıktan sonra yeni binary'nin zengin page projection'ı yazmasını, eski binary'nin yalnız `id/title/version` kolonlarını okuyup güncellemesini ve yeni alanları koruyan yeni binary okumasını SQLite'ta doğrular. `HYDRO_TEST_POSTGRES_URL` tanımlı CI koşulunda aynı contract PostgreSQL'de de çalışır.

`internal/release.RetainAssets`, release manifest'indeki `static/`, `assets/` ve `public/assets/` artifact'lerini checksum ile doğrulayıp ortak retention dizinine atomik olarak taşır; aynı yoldaki mevcut asset farklıysa değişiklik reddedilir ve `VerifyRetainedAssets` rollback penceresinde bütünlüğü tekrar kontrol eder.

`hydro upgrade check` migration çalıştırmadan config/profile, DB ping ve migration ledger checksum uyumluluğunu kontrol eder; başarılı preflight `upgrade-ready` çıktısı verir ve schema uyumsuzluğunda readiness açmaz.

Migration runner artık metadata'daki `transactional.<dialect>` değerine uyar. Transaction'lı ardışık migration'lar aynı transaction grubunda kalır; `transactional: false` migration kendi bağlantı yürütmesi ve ledger kaydıyla çalışır. SQLite `VACUUM` fixture'ı bu ayrımı contract seviyesinde doğrular. `HYDRO_TEST_POSTGRES_URL` tanımlı CI koşulunda `CREATE INDEX CONCURRENTLY` migration'ı aktif read transaction varken transaction dışında uygulanır ve indeks görünür hale gelir. Gerçek slot rollback'i hâlâ ayrı S06 tatbikatıdır.

`internal/release.Deployment` ADR-0005 sırasını local state machine olarak zorlar: Green hazırlanır, readiness alınır, retained asset warm-up tamamlanır, schema rollback safety onaylanır, sonra switch/drain/rollback yapılır. Bu sözleşme gerçek reverse-proxy trafiğini veya process restart'ını tek başına sertifikalamaz.

`TestSQLiteOnlineExpandAllowsConcurrentReadContract` WAL açık SQLite'ta aktif bir read transaction sürerken nullable expand migration'ın iki saniyelik deadline içinde tamamlandığını ve yeni kolonun görünür olduğunu doğrular. Bu deney migration'ın genel production lock süresini veya PostgreSQL DDL planını sertifikalamaz; gerçek blue/green trafik, yeni yazımdan sonra production rollback ve kapasite ölçümü S06/S07'nin açık kısmıdır.
