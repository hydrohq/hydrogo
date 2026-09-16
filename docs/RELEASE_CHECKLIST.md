# HydroGo release checklist

## Automated local gate

- [x] `scripts/verify-local.ps1` tamamlandı (module integrity, vuln scan, SBOM,
  vet, full/shuffled tests, fuzz, builds, generator smoke, health/readiness).
- [x] P0 task’larının ADR ve validation kanıtları mevcut.
- [x] Release manifest üretildi ve imzası ayrı anahtarla doğrulandı.
- [x] Migration SQL ve checksums gözden geçirildi; migrationguard temiz.
- [x] Demo/fixture verisi production image’ından ayrıldı.

Yerel kanıt: `scripts/verify-local.ps1` son olarak 2026-09-16 tarihinde
Windows üzerinde; `go mod verify`, `govulncheck`, full/shuffled/fuzz testleri,
SBOM, generator smoke ve health/readiness kontrolleri başarılıdır. Ubuntu/WSL2
karşı kanıtı [`security-check-2026-09-16.md`](validation/security-check-2026-09-16.md)
altındadır. Gerçek OCI build ise CI `container` job’ında çalışır.

## External gate (kanıt olmadan işaretlenmez)

- [x] PostgreSQL contract suite’i gerçek WSL PostgreSQL 18.6 servisiyle tamamlandı
  (`validation/R6-postgres-production-gates-2026-09-16.md`).
- [ ] Gerçek müşteri verisiyle temiz hedef restore, RPO/RTO ve rollback tatbikatı tamamlandı.
- [ ] Redis/Valkey failure/failover ve multi-instance rate limit testi yapıldı.
- [ ] Reverse-proxy blue/green, drain ve rollback drill tamamlandı.
- [ ] SMTP, object storage/CDN, ödeme sandbox ve queue credential’larıyla
  doğrulandı.
- [ ] Penetration test, legal/licensing ve müşteri ürün kararları kaydedildi.

## Sign-off

- Release/version:
- Evidence links:
- Known limitations:
- Approver and UTC timestamp:
