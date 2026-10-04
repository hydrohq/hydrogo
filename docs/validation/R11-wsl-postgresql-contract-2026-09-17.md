# R11 WSL PostgreSQL contract and restore evidence (2026-09-17)

## Environment

- Ubuntu WSL2, Go 1.27.0, linux/amd64
- PostgreSQL 18.6 (`Ubuntu 18.6-0ubuntu0.26.04.1`)
- Isolated database: `hydro_contract`
- Peer-authenticated WSL role: `testuser`

The test database is disposable and contains no customer data. Credentials are
not written to the repository or report.

## Commands and results

```bash
cd /path/to/project
export HYDRO_TEST_POSTGRES_URL='postgres://user@localhost:5432/hydro_contract?sslmode=disable'
go test ./tests/contract ./internal/backup -count=1 -p 1 -timeout=8m
# PASS: tests/contract
# PASS: internal/backup
```

The run covered the PostgreSQL content, news/taxonomy, site, media, auth,
transaction and migration contracts, plus PostgreSQL dump/restore and restore
control tests. The first run correctly rejected a non-privileged role for
temporary restore database creation; the isolated test role was then granted
`CREATEDB` and the backup suite passed. No production role was changed.

## Remaining production gate

This closes the local supported-version contract evidence only. A clean target
restore using customer-like data, measured RPO/RTO, external backup/key
recovery, and rollback after real writes remain production drills.
