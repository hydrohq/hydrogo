# Generated Edition PostgreSQL Validation — 2026-09-21

The checked-in Project and Article Edition migration metadata and the shared
SQLite/PostgreSQL contract suite were executed against the local WSL
PostgreSQL service. The Go process ran inside WSL so it could use the Unix
socket (Windows processes cannot address the WSL socket directly):

```text
cd /path/to/project
HYDRO_TEST_POSTGRES_URL="postgres://user@localhost:5432/hydrogo_backup_cert2" \
  go test ./tests/contract ./internal/generator ./internal/modules -count=1 -timeout=10m

ok  github.com/ardaozyildirim/hydrogo/tests/contract
ok  github.com/ardaozyildirim/hydrogo/internal/generator
ok  github.com/ardaozyildirim/hydrogo/internal/modules
```

This evidence covers the real PostgreSQL adapter paths for module isolation,
transaction and migration compatibility, tenant/RLS, auth, pages/news/site,
backup-related contracts, and the generated-edition migration metadata checks.
The disposable database is not a production certification target; a release
pipeline must still run the same suite against its managed PostgreSQL version.
