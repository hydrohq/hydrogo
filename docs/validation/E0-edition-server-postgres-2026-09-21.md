# Client Edition PostgreSQL Server Smoke — 2026-09-21

The Project Edition standalone binary was run inside WSL against the local
PostgreSQL service. The operator migration subcommand and the live generated
HTTP handler both completed successfully:

```text
HYDRO_TEST_POSTGRES_URL=postgres://user@localhost:5432/hydrogo_backup_cert2
edition-server migrate --dialect postgres --dsn "$HYDRO_TEST_POSTGRES_URL"
migration-applied edition=project dialect=postgres

GET /healthz -> 200
POST /api/v1/project -> 201 {"id":"p-pg", ...}
GET /api/v1/project/p-pg -> 200 {"id":"p-pg", ...}
```

The server process was stopped after the smoke. This is disposable local
evidence; release certification must repeat it against the managed PostgreSQL
major version and a production Edition policy with tenant/RBAC binding.
