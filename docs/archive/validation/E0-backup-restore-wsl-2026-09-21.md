# E0 Backup/Restore WSL Validation — 2026-09-21

SQLite backup/restore contracts pass locally:

```text
go test ./internal/backup -count=1
```

The PostgreSQL dump/restore contract now passes against a fresh WSL database
(`HYDRO_TEST_POSTGRES_URL=postgres://user@localhost:5432/hydrogo_backup_cert2`):

```text
HYDRO_TEST_POSTGRES_URL=... go test ./internal/backup -count=1
ok  github.com/ardaozyildirim/hydrogo/internal/backup
```

The same WSL DSN also passed the combined backup and contract suite:

```text
go test ./internal/backup ./tests/contract -count=1 -timeout=5m
ok  github.com/ardaozyildirim/hydrogo/internal/backup
ok  github.com/ardaozyildirim/hydrogo/tests/contract
```

The test temporarily removes `FORCE ROW LEVEL SECURITY` on the disposable
source database only; this separates snapshot mechanics from the dedicated
RLS isolation contract. A production drill with an ordinary role is still
rejected at `pg_dump`, as intended. HydroGo accepts an explicitly configured
`HYDRO_PG_PRIVILEGED_ROLE` and passes it to `pg_dump`/`pg_restore`; it never
silently escalates credentials. Production certification requires a dedicated
backup role with the reviewed RLS-bypass privilege and a clean target restore.
