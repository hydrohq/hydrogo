# R106 — WSL PostgreSQL restore ve Redis race drills

**Tarih:** 2026-09-22  
**Durum:** PASS (disposable WSL servisleri)

İki disposable servis tatbikatı çalıştırıldı:

- `scripts/verify-wsl-postgres-drill.ps1`: yeni PostgreSQL database oluşturma,
  backup/restore sözleşmesi ve `tests/contract` suite; sonuç PASS. Database
  tatbikat sonunda silindi.
- `scripts/verify-wsl-redis-drill.ps1`: geçici Redis 6xxx portunda atomic
  admission/race testi (`go test -race ./internal/security`); sonuç PASS.

Redis yalnızca bellek overcommit ayarı için operasyonel uyarı verdi; test
başarısını etkilemedi. Bu rapor production sağlayıcısı, reverse-proxy rollback,
harici key escrow veya gerçek müşteri RPO/RTO kapasite ölçümünün yerine geçmez.
