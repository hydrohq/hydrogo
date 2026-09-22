# R105 — WSL PostgreSQL/Redis production-gate preflight

**Tarih:** 2026-09-22  
**Commit:** `95fce7a76323b5ca5af31473bba770d3f13a0683`  
**Durum:** PASS (WSL2 yerel servisler)

`scripts/verify-production-gates.sh`, PostgreSQL 18 ve Redis/Valkey 8
üzerinde aşağıdaki kapıları geçti:

- gofmt ve `go mod verify`
- workflow action pin kontrolü
- govulncheck (kodun çağırdığı sembollerde açık yok; transitif kullanılmayan
  bağımlılık uyarıları araç çıktısında ayrıca belirtildi)
- `go vet ./...`
- tam ve shuffle edilmiş test suite
- race-sensitive paketler
- reproducible HydroGo binary build
- portable SQLite benchmark
- Redis/Valkey atomic limiter + TTL testi

Bu kanıt gerçek boş hedefe PostgreSQL restore, iki süreçli reverse-proxy
blue/green rollback, harici backup/key recovery, üretim anahtar seremonisi ve
müşteri RPO/RTO/kapasite ölçümünü kapsamaz. Gate runner bunları bilerek
`EXTERNAL DRILLS STILL REQUIRED` olarak bırakır.
