# R51 — WSL PostgreSQL + Redis production-gate preflight

**Date:** 2026-09-17  
**Commit:** `c7eb10ff745363fecee184508146aa380593a0a3`  
**Status:** Başarılı (yerel servis kanıtı)

Ubuntu/WSL2 üzerinde çalışan PostgreSQL 18.6 ve Redis/Valkey servisine karşı
`scripts/verify-production-gates.sh` çalıştırıldı. Test DSN'i yalnızca yerel
Unix socket üzerinden kullanıldı; rapora parola veya token yazılmadı.

Geçen kapılar:

- gofmt, module integrity, workflow action pin ve govulncheck
- go vet, full test suite ve shuffled test suite
- race-sensitive package suite (PostgreSQL contract testleri dahil)
- reproducible HydroGo build
- SQLite portable benchmark
- Redis/Valkey atomic limiter + TTL testi

Runner raporu: `.tmp/gates-wsl/production-gates-20260917-155008.md`.

Bu kanıt gerçek müşteri verisiyle restore, reverse-proxy blue/green/rollback,
harici backup/key recovery, production key ceremony veya müşteri kapasite/RPO/RTO
tatbikatlarının yerine geçmez; bunlar açık production kapılarıdır.
