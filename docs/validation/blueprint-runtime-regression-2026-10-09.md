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
No PostgreSQL test ran in this environment: `HYDRO_TEST_POSTGRES_URL` is
unset, and the Docker API is inaccessible (`permission denied` on
`/var/run/docker.sock`). Earlier PostgreSQL evidence remains in the existing
generated Admin validation reports; this run does not refresh or broaden it.
The authenticated Chromium generated Admin flow remains unverified here
because the fixture cannot bind loopback; see
`generated-block-editor-theme-matrix-2026-10-09.md` for the external-fixture
command path.
