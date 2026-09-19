# Validation Report R25: PostgreSQL Tenant RLS Contract Isolation Certification

- Date: 2026-09-20
- Scope: PostgreSQL Row-Level Security Contract Certification
- ADR: [ADR-0185](../architecture/0185-postgres-rls-contract-certification.md)
- Status: PASS

## Objectives & Verifications

1. **Table Coverage Verification:** Certified that all 29 tenant-scoped PostgreSQL tables have `relrowsecurity=true` and `relforcerowsecurity=true` configured, ensuring RLS cannot be bypassed even by table owners.
2. **Policy Existence:** Certified that `hydrogo_tenant_isolation` exists on all 29 tables.
3. **Cross-Tenant Isolation Contract:**
   - Certified read isolation: queries executed under `tenant-beta` context return 0 rows for `tenant-alpha` data, even when application `WHERE tenant_id` predicates are omitted.
   - Certified update isolation: direct updates targeting `tenant-alpha` from a `tenant-beta` transaction modify 0 rows.
   - Certified delete isolation: direct deletes targeting `tenant-alpha` from a `tenant-beta` transaction delete 0 rows.
   - Certified insert isolation: `WITH CHECK` rejects inserting records with mismatched `tenant_id`.
4. **Fail-Closed Verification:** Certified that transactor invocations with missing or empty tenant contexts fail closed immediately.
