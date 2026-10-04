# R16 WSL PostgreSQL module-isolation contract — 2026-09-17

## Environment

- WSL Ubuntu, Go 1.27.0
- PostgreSQL 18.6 local service
- Database: disposable `hydro_contract`, role `testuser`
- Revision under test: `5ed79a6`

## Command and result

```bash
HYDRO_TEST_POSTGRES_URL='postgres://user@localhost:5432/hydro_contract?sslmode=disable' \
  go test ./tests/contract \
  -run '^TestPostgresModuleFilteringLeavesInactiveModuleSchemaAbsent$' \
  -count=1 -p 1 -timeout=5m
```

Result: **PASS** (0.145 s).

The contract applies the `pages + media` migration set in an isolated schema,
confirms the required tables exist, confirms Commerce and News tables (including
News taxonomy) are absent, and runs the filtered migration compatibility check.
The schema is dropped after the test.

## Limitations

This proves the generic PostgreSQL migration-filtering boundary. It does not
replace a full customer-data restore, multi-profile capacity run, or provider
failover drill. Per-preset certification and production deployment evidence
remain release gates.
