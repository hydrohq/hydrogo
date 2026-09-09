# R3 yerel PostgreSQL 18.6 contract ve restore doğrulaması

**Tarih:** 2026-09-09  
**Commit:** `7d557324d50e010146580c167d22f9b227845044`  
**Durum:** Başarılı; yerel contract kanıtı, ticari üretim sertifikasyonu değil.

## Ortam

- PostgreSQL 18.6, Windows x86_64, `127.0.0.1:5432`
- Go toolchain: repository CI ile aynı `go 1.27.0` hedefi
- Test veritabanı: contract testleri için ayrılmış yerel veritabanı
- PostgreSQL dump/restore araçları: PostgreSQL 18 istemci araçları

## Çalıştırılan kapılar

```powershell
$env:HYDRO_TEST_POSTGRES_URL = "postgres://...@127.0.0.1:5432/postgres?sslmode=disable"
go test ./internal/backup -run '^TestPostgresRestore' -count=1 -v
go test ./tests/contract -run '^TestPostgres' -count=1 -v
```

Başarılı kanıtlar:

- Restore sonrası session iptali, egress quarantine ve eski yazma fence'i.
- Quarantine kaldırılmadan job claim yapılamaması; kontrollü açılıştan sonra claim yapılabilmesi.
- Pages/media erasure replay idempotency.
- Auth, media, News, Pages, Site ve transaction contract'ları.
- PostgreSQL concurrent migration runner ve long-snapshot `CREATE INDEX CONCURRENTLY` davranışı.

## Sınırlar

Bu çalışma gerçek bir müşteri veritabanında yapılmadı ve dış backup provider, key escrow, reverse proxy/process restart, production signing key ceremony veya müşteri RPO/RTO ölçümü kanıtlamaz. S07 satış sertifikasyonu ve temiz, harici hedefe restore tatbikatı açık kalır.
