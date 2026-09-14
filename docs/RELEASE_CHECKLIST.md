# HydroGo release checklist

## Automated local gate

- [ ] `scripts/verify-local.ps1` tamamlandı (module integrity, vuln scan, SBOM,
  vet, full/shuffled tests, fuzz, builds, generator smoke, health/readiness).
- [ ] P0 task’larının ADR ve validation kanıtları mevcut.
- [ ] Release manifest üretildi ve imzası ayrı anahtarla doğrulandı.
- [ ] Migration SQL ve checksums gözden geçirildi; migrationguard temiz.
- [ ] Demo/fixture verisi production image’ından ayrıldı.

## External gate (kanıt olmadan işaretlenmez)

- [ ] PostgreSQL contract/restore testi gerçek servisle tamamlandı.
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
