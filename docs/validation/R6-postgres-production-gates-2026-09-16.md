# PostgreSQL production-gate kanıtı (2026-09-16)

WSL Ubuntu üzerinde çalışan PostgreSQL 18.6 ile izole `hydro_gate_local`
veritabanı kullanıldı. Bağlantı peer authentication üzerinden yapıldı;
herhangi bir parola veya secret rapora yazılmadı. Geçici veritabanı test sonrası
silinmelidir.

## Komut

```bash
cd /mnt/c/dev/cms
HYDRO_TEST_POSTGRES_URL="postgres://arda@/hydro_gate_local?host=/var/run/postgresql" \
  HYDRO_GATE_REPORT=.tmp/gates/postgres-local-20260916-full.md \
  bash scripts/verify-production-gates.sh
```

## Sonuç

Tam akış **PASS**:

- gofmt, module integrity, workflow action pins, govulncheck ve go vet;
- SQLite/PostgreSQL dâhil tam ve shuffled test suite;
- race-sensitive paketler;
- iki kez aynı hash üreten reproducible HydroGo build;
- portable benchmark (`10.3 ms/op`, 10 iterasyon).

Bu çalışma PostgreSQL bağlantı/kontrat ve yerel otomatik gate kanıtıdır. Gerçek
müşteri verisiyle temiz hedefe restore, proxy blue/green rollback, harici
backup/key recovery ve kapasite/RPO/RTO tatbikatları hâlâ production kapısıdır.
