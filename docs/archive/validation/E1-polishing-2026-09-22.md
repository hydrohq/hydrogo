# E1 Polishing Validation — 2026-09-22

## Evidence

- `go test ./...` passed (Windows workspace).
- `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/html-comment-lint.ps1` passed.
- `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-local.ps1` full local gate was executed: module integrity, govulncheck, SBOM, asset/container/secret lint, vet, and full test suite steps passed.
- Endpoint metrics tests verify route normalization, p50/p95/p99, status/timeout/rate-limit, bytes, and tenant/module dimension contracts.
- Analytics SQLite contract tests verify consent gate, aggregate upsert, bounded queue, and report filters.
- Backup tests verify SQLite/PostgreSQL snapshot/restore staging, manifest digest, and path traversal guards.
- `scripts/verify-wsl-postgres-drill.ps1` passed on WSL PostgreSQL 18 by running backup and
  contract suites sequentially; the disposable database was dropped at the end.
- OTLP profile unit tests verify endpoint and 0..1 sampling validation; the optional
  `InstallOTLPProvider` path verifies HTTP OTLP batch exporter setup.

## Deliberate Operational Boundaries

OTLP exporter selection is left to the deployment provider; the Compact profile operates
without an exporter. KMS/key escrow is modeled purely as a manifest reference and operator boundary;
the key is not retained within HydroGo. `templ generate` and `go mod tidy` are not scaffold hooks.
