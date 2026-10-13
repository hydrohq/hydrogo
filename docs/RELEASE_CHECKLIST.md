# HydroGo release checklist

## Automated local gate

- [x] `scripts/verify-local.ps1` completed (module integrity, vuln scan, SBOM,
  vet, full/shuffled tests, fuzz, builds, generator smoke, health/readiness).
- [x] P0 task ADRs and validation evidence in place.
- [x] Release manifest generated and verified with detached signature key.
- [x] Migration SQL and checksums reviewed; migrationguard clean.
- [x] Demo/fixture data isolated from production image.

Local evidence: `scripts/verify-local.ps1` was verified on 2026-09-16
on Windows: `go mod verify`, `govulncheck`, full/shuffled/fuzz tests,
SBOM, generator smoke, and health/readiness checks passed. Ubuntu/WSL2
counterpart evidence is documented under [`security-check-2026-09-16.md`](validation/security-check-2026-09-16.md).
The real OCI build runs in the CI `container` job.

## External gate (do not mark without evidence)

- [x] PostgreSQL contract suite verified against live WSL PostgreSQL 18.6 service
  (`validation/R6-postgres-production-gates-2026-09-16.md`).
- [ ] Clean target restore, RPO/RTO, and rollback rehearsals completed with real customer data.
- [ ] Redis/Valkey failure/failover and multi-instance rate limiting tested.
- [ ] Reverse-proxy blue/green, connection drain, and rollback drill completed.
- [ ] Verified with SMTP, object storage/CDN, payment sandbox, and queue credentials.
- [ ] Penetration test, legal/licensing review, and customer product sign-offs recorded.

## Sign-off

- Release/version:
- Evidence links:
- Known limitations:
- Approver and UTC timestamp:
