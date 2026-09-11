# Initial Support Matrix Evidence

This table is not a commercial support SLA; a profile is not declared supported until the CI or contract evidence next to each row is fully completed.

| Area | CI / Contract Evidence | Status |
|---|---|---|
| Go toolchain | Go 1.27.0 build/test/vet on Linux and Windows | Verifying |
| SQLite | `modernc.org/sqlite` v1.58.0; migration, rollback, export/import contracts | Technical contract ready |
| PostgreSQL | PostgreSQL 18 service; migration, transaction, restore, and export/import contracts | Technical contract ready |
| PostgreSQL backup tools | PostgreSQL 18 `pg_dump`/`pg_restore`; custom-format dump/restore round-trip | Technical contract ready |
| Linux amd64 | `ubuntu-latest` CI build/test and reproducible binary | CI verified |
| Windows amd64 | `windows-latest` CI build/test and reproducible binary | CI verified |
| Compact profile | Typed budget and `hydro doctor`; real customer benchmark | Open |
| Standard profile | Typed budget; real PostgreSQL pool/throughput benchmark | Open |
| Distributed profile | Architectural contract exists; multi-instance/L2/S3/CDN drill pending | Open |
| Modules (Pages & News) | SQLite & PostgreSQL contract tests, admin/API flows | Technical contract ready (Pilot candidate) |
| Module (Commerce) | Domain/SQL store/checkout contract tests, webhook & sandbox provider tests | Experimental / Pilot candidate (Prod certification open) |

Version upgrades, provider support, and operating system support are officially locked with a separate release decision only after real benchmarks, clean target restores, and release/deprecation schedules are completed.
