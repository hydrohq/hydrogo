# Admin Studio Targeted Validation — 2026-10-09

## Result

Tests for Admin Studio, Blueprint generator, Edition composition, and security packages
passed. The initial full `go test ./...` attempt hit loopback listener restrictions
in non-Admin packages. HTTP contracts that do not require a listener were migrated to in-memory
RoundTripper, and HTTP/FCGI server lifecycle tests were migrated to a `net.Pipe` listener.
The final full repository run and `go vet ./...` passed.

## Passed Validations

Using the correct temporary test directory (`/tmp/hydrogo-admin-tests`) and an external Go cache:

```sh
GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./internal/adminstudio ./internal/collections/... ./internal/mailer/... \
  ./internal/media/... ./internal/push/... ./internal/trash/... ./internal/webhooks/... \
  ./internal/ui ./tests/contract ./examples/article-edition/custom \
  ./examples/project-edition/custom ./internal/generator ./internal/auth \
  -count=1 -p 1 -timeout=5m
```

In this command, the only test causing package failure was the loopback-bound
Social OAuth test in `internal/auth`. Admin/role/permission tests also passed
in a narrower scope. Following this separation:

```sh
GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./internal/auth -run 'TestPermissionCatalog|TestAdminUserManagementMutationDeniesExplicitPermissionOverride|TestAdminCannotChangeOwnRoleOrPermissions|TestAdminUserMutationEndpointsRejectEditorAndViewer|TestAdminPermissionEndpoint' \
  -count=1 -p 1 -timeout=3m

GOCACHE="$PWD/.cache/admin-test" TMPDIR=/tmp/hydrogo-admin-tests GOPROXY=off \
  go test ./cmd/hydro -run 'Test(Admin|Resource|ProjectCheck|RunResource|RunInteractiveResource|ResourceWizard|InteractiveResource|EveryRegisteredAdminControlRoute|RunPermissionsCheck)' \
  -count=1 -p 1 -timeout=5m
```

Both focused test commands passed. `go vet` and `git diff --check` passed for the
relevant package set as well. Two Edition composition tests, Generated Admin HTTP
tests, and SMTP tests now run without opening listeners; using an external `TMPDIR`
in generator tests prevents temporary modules from accidentally inheriting the root repo module.

## Remaining Runtime Environment Gates

- The Admin responsive Playwright test was initiated by directing artifact output to a
  writable directory. Go setup and SQLite migrations succeeded, but the fixture
  received `listen EPERM` when starting the `127.0.0.1` test server. No new browser
  screenshots, axe results, or touch-target measurements were produced.
- `HYDRO_TEST_POSTGRES_URL` is not set. Docker API access was denied; results from the local
  cluster start attempt are recorded below. Prior PostgreSQL evidence does not count as new
  live PostgreSQL results for this iteration.

### PostgreSQL Local-Start Retry

A temporary PostgreSQL 18.6 cluster was successfully created under `/tmp` via `initdb`.
The attempt to start it via `pg_ctl` strictly using a Unix-domain socket stopped with
an `Operation not permitted` error while PostgreSQL opened the socket file; Docker API
also returned `permission denied`, and no external PostgreSQL DSN was configured.
Consequently, this attempt did not run migrations or contract tests; the live PostgreSQL
validation gate remains open.

## Re-Validation in this Iteration

Following listener errors observed during the initial full suite run, AI, OAuth,
Sentry, link checker, reverse proxy, SDK broker/download, BFF, and public site
transport tests now use in-memory HTTP transports. HTTP server graceful shutdown and
FastCGI tests exercise identical framing and drain behavior using listeners backed by
`net.Pipe` without opening real sockets. Production HTTP transport and listening behavior
remain unchanged; only the server's internal routine accepting listeners was made testable.

```sh
GOPROXY=off GOCACHE="$PWD/.cache/admin-test" GOTMPDIR=/tmp/hydro-go-home TMPDIR=/tmp \
  go test ./... -count=1 -p 1 -timeout=8m
GOPROXY=off GOCACHE="$PWD/.cache/admin-test" GOTMPDIR=/tmp/hydro-go-home TMPDIR=/tmp \
  go vet ./...
```

Both commands passed. Playwright and PostgreSQL gates noted above remain open.
