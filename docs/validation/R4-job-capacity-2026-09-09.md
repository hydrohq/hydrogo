# R4 durable job queue kapasite doğrulaması

## Kapsam

Durable SQL queue'nun `MaxEntries` kontrolü artık producer transaction'ı içinde
singleton kapasite satırını güncelleyerek serialize edilir. PostgreSQL'de satır
kilidi, SQLite'ta writer serialization aynı kapasite kontrolünü korur. Yeni
`202609090020_job_capacity_lock` migration'ı her dialect için aynı lock satırını
oluşturur ve idempotent biçimde seed eder.

## Kanıt

- Commit: `c8f4104`
- Ortam: Windows geliştirici ortamı
- Komutlar:
  - `go test ./internal/jobs/sqlstore -count=5`
  - `go test ./... -p 1`
  - `go vet ./...`
  - `git diff --check`
- Sonuç: tüm yerel testler ve statik kontroller başarılı.
- Contract: `TestSQLiteQueueCapacityIsBoundedForConcurrentProducers`, iki eşzamanlı
  producer ile kapasite `1` iken yalnız bir işin commit edilebildiğini ve claimable
  iş sayısının sınırı aşmadığını doğrular.

## Sınırlar

Bu turda `HYDRO_TEST_POSTGRES_URL` tanımlı olmadığından PostgreSQL concurrency
contract'ı yerel çalıştırılmadı; CI/production gate tarafından çalıştırılmalıdır.
Bu kanıt ayrıca çoklu müşteri donanımı veya harici broker kapasitesini
sertifikalamaz.
