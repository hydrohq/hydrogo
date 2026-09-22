# E1 Polishing validation — 2026-09-22

## Kanıt

- `go test ./...` geçti (Windows workspace).
- `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/html-comment-lint.ps1` geçti.
- `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-local.ps1` full local gate'i çalıştırıldı: module integrity, govulncheck, SBOM, asset/container/secret lint, vet ve full test suite adımları geçti.
- Endpoint metrics testleri route normalization, p50/p95/p99, status/timeout/rate-limit, bytes ve tenant/module dimension sözleşmesini doğrular.
- Analytics SQLite contract testleri consent gate, aggregate upsert, bounded queue ve report filtrelerini doğrular.
- Backup testleri SQLite/PostgreSQL snapshot/restore staging, manifest digest ve path traversal guard'larını doğrular.
- `scripts/verify-wsl-postgres-drill.ps1` WSL PostgreSQL 18 üzerinde backup ve
  contract paketlerini sıralı çalıştırarak geçti; disposable database sonunda
  silindi.
- OTLP profile unit testleri endpoint ve 0..1 sampling doğrulamasını; optional
  `InstallOTLPProvider` path'i HTTP OTLP batch exporter kurulumunu doğrular.

## Bilinçli operasyon sınırları

OTLP exporter seçimi deployment provider'ına bırakılmıştır; Compact profile exporter
olmadan çalışır. KMS/key escrow yalnızca manifest referansı ve operator sınırı
olarak modellenmiştir; anahtar HydroGo içinde tutulmaz. `templ generate` ve `go
mod tidy` scaffold hook'u değildir.
