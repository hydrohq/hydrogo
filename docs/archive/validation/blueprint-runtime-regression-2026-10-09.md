# Blueprint runtime regression revalidation — 2026-10-09

This re-runs the native Blueprint/Admin activation path and the independently
compiled Edition contract after the recent Admin security changes.

## Passed

- `go test ./internal/generator -count=1 -p 1 -timeout=5m`
- `go vet ./internal/generator`
- `go test ./examples/project-edition/... -count=1 -p 1 -timeout=5m`
- `go vet ./examples/project-edition/...`
- In the separate module, `go test -mod=mod ./... -count=1 -p 1 -timeout=5m`
  and `go vet -mod=mod ./...` in `tests/external-edition`
- `hydro project check` on `examples/project-edition`: `valid profile=compact`
  with SQLite and the expected active modules.
- `moduleguard`, `migrationguard` (110 migrations), and the external Edition
  `importguard` passed.

## Limits

The independent Edition tests cover its SQLite persistence and route contracts.
No PostgreSQL driver/adapter contract test ran in this environment:
`HYDRO_TEST_POSTGRES_URL` is unset, and the Docker API is inaccessible
(`permission denied` on `/var/run/docker.sock`). Earlier PostgreSQL evidence
remains in the existing generated Admin validation reports; this run does not
refresh or broaden it.

## PostgreSQL 18.6 standalone DDL syntax check — 2026-10-09

PostgreSQL 18.6 server binaries are installed. A fresh disposable cluster
initialized successfully, but the server could not bind either a TCP listener
or a Unix socket (`Operation not permitted`), so no application driver could
connect. The PostgreSQL standalone backend (which does not open a listener)
accepted the generated `article`, `milestone`, and `content_block` migration
SQL, and a catalog query found all three tables in `public`:

```sh
HYDROGO_PGDATA_TMP=/tmp/hydrogo-admin-pgdata-e231
initdb -D "$HYDROGO_PGDATA_TMP" -U hydrogo_test --auth=trust --no-locale --encoding=UTF8
sed '/^--/d' examples/project-edition/generated/article/migration.sql \
  examples/project-edition/generated/milestone/migration.sql \
  examples/project-edition/generated/content_block/migration.sql \
  | tr '\n' ' ' \
  | postgres --single -D "$HYDROGO_PGDATA_TMP" postgres
printf "SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename IN ('articles','milestones','content_blocks') ORDER BY tablename;\n" \
  | postgres --single -D "$HYDROGO_PGDATA_TMP" postgres
```

This verifies PostgreSQL SQL parsing/catalog creation for those additive DDL
files only. It does not exercise the PostgreSQL wire protocol, Go adapter,
transaction/concurrency behavior, migration runner, tenant isolation, or
rollback. The migration metadata remains forward-only/restore-based as
documented; those runtime gates remain open.

### Standalone schema and transaction semantics

Catalog queries on the same temporary cluster confirmed each table has `id`
as its primary key and all generated columns are PostgreSQL `text NOT NULL`.
An article insert succeeded; an update inside `BEGIN` followed by `ROLLBACK`
left the original title unchanged. A `NULL` title insert raised PostgreSQL's
not-null violation and a follow-up count remained zero. These are direct
PostgreSQL engine semantics in single-user mode, not tests through HydroGo's
database adapter, migration runner, or HTTP/API stack.

```sh
HYDROGO_PGDATA_TMP=/tmp/hydrogo-admin-pgdata-e231
printf "INSERT INTO articles(id, body, title) VALUES ('e231-rollback-check', 'body', 'before');\nBEGIN; UPDATE articles SET title='after' WHERE id='e231-rollback-check'; ROLLBACK;\nSELECT title FROM articles WHERE id='e231-rollback-check';\n" \
  | postgres --single -D "$HYDROGO_PGDATA_TMP" postgres
printf "INSERT INTO articles(id, body, title) VALUES ('e231-null-check', 'body', NULL);\nSELECT count(*) FROM articles WHERE id='e231-null-check';\n" \
  | postgres --single -D "$HYDROGO_PGDATA_TMP" postgres
```
The authenticated Chromium generated Admin flow remains unverified here
because the fixture cannot bind loopback; see
`generated-block-editor-theme-matrix-2026-10-09.md` for the external-fixture
command path.
