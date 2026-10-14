# Site not-found PostgreSQL RLS correction — 2026-10-09

## Root cause

`202609290008_site_not_found_hits/postgres.sql` forced row-level security but
bound the policy to `hydrogo.tenant_id`. The shared tenant transaction
contract binds `app.tenant_rls_enforced` and `app.current_tenant_id` instead.
As a result, the PostgreSQL Site counter repository could not persist a
missing-path hit when using the supported tenant transaction. SQLite tests
passed because they exercise repository predicates, not PostgreSQL RLS.

## Correction

Added `202610090001_site_not_found_rls_guc`, a forward-only, transactional
PostgreSQL migration that replaces the policy expression with the same
transaction-local tenant settings as the core RLS contract. It preserves the
table, policy name, forced RLS, indexes and stored counters. SQLite and MySQL
migrations are checksum-protected no-ops; repository predicates remain their
tenant boundary. Architecture decision and rollout constraints are in
[ADR-0366](../architecture/0366-site-not-found-tenant-rls-binding.md).

## Verification

The migration and tenant isolation ran on PostgreSQL 18.6 as `arda`, verified
`NOSUPERUSER` and `NOBYPASSRLS`, in an isolated test database/schema. The
existing shared-tenant repository contract writes, reads, counts, and deletes
the same path independently for two tenants. A registered Site appearance
route test also verified that invalid CSRF and `site.manage=false` produce no
database write, while an authorized Admin saves values read back from
PostgreSQL.

```sh
HYDRO_TEST_POSTGRES_URL='postgres://<role>@/<empty-test-database>?host=/run/postgresql&sslmode=disable' \
  go test ./modules/site/... -count=1 -p 1 -timeout=5m
HYDRO_TEST_POSTGRES_URL='postgres://<role>@/<empty-test-database>?host=/run/postgresql&sslmode=disable' \
  go vet ./modules/site/...
go run ./tools/migrationguard -root migrations
```

All Site packages passed with PostgreSQL configured (SQLite contracts also ran
in the same package suite); vet passed; migrationguard passed with 111
migrations. This is local database/HTTP evidence, not a customer deployment or
production identity test. No production migration has been applied.
