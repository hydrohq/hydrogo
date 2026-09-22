# R101 — Admin bulk ve generated tenant RLS doğrulaması

**Tarih:** 2026-09-22  
**Durum:** Yerel doğrulama başarılı; gerçek PostgreSQL restore/capacity kabulü
bu raporun kapsamı dışındadır.

## Değişiklikler

- `adminstudio.ExecuteBulk`: 500 kayıt hard limitli, deterministik sıralı,
  callback hata metnini sızdırmayan ve context iptalinde açıkça tamamlanmamış
  sonuç üreten orchestration primitive'i.
- `tenant.VerifyProvisioningRLSForTables`: Edition’ın generated tenant tablo
  listesini fail-closed RLS doğrulamasına bağlayan API.
- Generated tenant kaynaklarında explicit tenant parametreleri, migration
  `(tenant_id, id)` index'i ve RLS verifier release checklist'i.

## Çalıştırılan kontroller

```text
go test ./... -p 1 -count=1 -timeout=5m   PASS
go vet ./...                              PASS
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...  PASS
go test ./internal/generator -run TestGeneratedTenantScopedResourceBindsTenantContract -v  PASS
go test ./internal/tenant ./internal/adminstudio -count=1  PASS
scripts/html-comment-lint.ps1            PASS
scripts/adr-drift-check.ps1              PASS (259 ADR)
```

## Sınırlar

`go mod verify` Windows ortamında modül doğrulama aşamasında tamamlanmadığı
için bu raporda başarılı kabul edilmedi. PostgreSQL üzerinde gerçek policy
kataloğu, reverse-proxy blue/green, temiz ortama restore, key ceremony ve
müşteri kapasite/RPO/RTO ölçümleri ayrıca yürütülmelidir.
