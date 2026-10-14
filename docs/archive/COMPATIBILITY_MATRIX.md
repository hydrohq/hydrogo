# HydroGo Compatibility Matrix

This table defines the proven boundaries of current R3/R4 contracts. The term "Compatible" indicates behavior verified through corresponding contract tests and migration checksum validation; unlisted or unsupported combinations do not constitute a sales guarantee.

| Area | Epoch / Identity | Old → New Deploy | New → Old Deploy | Evidence |
|---|---|---|---|---|
| Core runtime/config | Config schema current version; unknown fields denied | New binary reads old config; defaults validated if no new required fields | Old binary operates only if it encounters no unknown routes/modules | config/router tests |
| Capacity profile | `compact`, `standard`, `distributed`; bounded budgets | New binary defaults to Compact if profile is omitted; enforces pool/cache/queue/media bounds when configured | Old binary ignores profile field and runs with fixed built-in budgets | capacity/config contracts; live benchmark open |
| Module manifest | `extension.APIStatus`, edition import boundary | New edition attaches to old Core slot via public extension contracts only | Old edition does not load new News/Site module | Client/News importguard CI |
| Schema/migration | `schema_migrations` SHA-256; 16 migrations | New binary applies migrations first, then validates checksum compatibility | Old binary reads and updates existing page data without using new columns | migration restart/checksum + rollback compatibility contracts |
| Pages/content | revision + optimistic `version`; published projection | New revision readable alongside old published projection | Old binary does not see new draft revisions, preserves published fields | Pages SQLite/PostgreSQL contracts |
| Site/SEO/forms | Site Edition migrations 009–011; form retention | New routes receive no traffic on old binary; publish projection preserved | Old binary does not write to new navigation/form tables | Site contracts + runbook |
| News/events | revision + `schedule_generation`; event id contains action/version | New scheduler safely skips old drafts via generation check | Old binary ignores scheduled news tables; unaffected if news route is inactive | News schedule/worker contracts |
| Jobs | migration 013; lease token + expiry | New worker claims and handles old queued rows via lease | Old in-process worker does not see SQL job table | SQL job store SQLite contract |
| Assets/media | immutable storage key, pending → ready | New binary serves old ready assets with private/public metadata | Old binary does not delete retained assets | media upload/backup contracts |

## Open Verifications

- Upgrade verification from migration 008 to the complete schema using `tests/contract/testdata/legacy_customer.json`; verified preservation of published page projections, media metadata, and auth hashes.
- PostgreSQL service matrix run for SQL job leasing is pending.
- PostgreSQL online DDL lock measurement, retained asset verification after rollback, and live Blue/Green traffic drills remain open under S06/S07 gates; SQLite expand/read and dual-dialect old binary rollback read-write contracts are verified.
