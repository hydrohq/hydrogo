# Validation Report R25: PostgreSQL Tenant RLS Contract Isolation Certification

- Date: 2026-09-20
- Scope: PostgreSQL Row-Level Security Contract Certification
- ADR: [ADR-0185](../architecture/0185-postgres-rls-contract-certification.md)
- Status: CODE CONTRACT READY (Automated test SKIPPED in local runner without live PostgreSQL; requires HYDRO_TEST_POSTGRES_URL)

## Objectives & Verifications

1. **Table Coverage Verification:** Formulated schema verification covering all 29 tenant-scoped PostgreSQL tables for `relrowsecurity=true` and `relforcerowsecurity=true`.
2. **Policy Existence:** Certified definition of `hydrogo_tenant_isolation` on all 29 tables.
3. **Cross-Tenant Isolation Contract (`tests/contract/postgres_rls_contract_test.go`):**
   - Corrected SQL fixtures for `site_settings` and `members` schemas.
   - Designed read isolation: queries executed under `tenant-beta` context return 0 rows for `tenant-alpha` data.
   - Designed update isolation: direct updates targeting `tenant-alpha` from a `tenant-beta` transaction modify 0 rows.
   - Designed delete isolation: direct deletes targeting `tenant-alpha` from a `tenant-beta` transaction delete 0 rows.
   - Designed insert isolation: `WITH CHECK` rejects inserting records with mismatched `tenant_id`.
4. **Execution Status:** Contract test in `tests/contract/postgres_rls_contract_test.go` requires a live PostgreSQL instance via `HYDRO_TEST_POSTGRES_URL`. When the environment variable is not configured, the test skips by design. Full physical certification is pinned to the real PostgreSQL preflight gate.
