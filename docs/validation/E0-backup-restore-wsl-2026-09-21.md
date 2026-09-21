# E0 Backup/Restore WSL Validation — 2026-09-21

SQLite backup/restore contracts pass locally:

```text
go test ./internal/backup -count=1
```

The PostgreSQL drill reached the `pg_dump` boundary and was intentionally
rejected when run with a non-privileged role while `FORCE ROW LEVEL SECURITY`
was active. HydroGo now accepts an explicitly configured
`HYDRO_PG_PRIVILEGED_ROLE` and passes it to `pg_dump`/`pg_restore`; it never
silently escalates credentials. Production certification requires a dedicated
backup role with the reviewed RLS-bypass privilege and a clean target restore.
