# Admin Studio full Go regression — 2026-10-09

The full root-module Go suite passed after the generated Admin route, CSRF,
role, generator, and error-recovery changes:

```sh
GOCACHE=/home/arda/hydrogo/.cache/admin-commit-work/.cache/go-build \
GOTMPDIR=/tmp/hydrogo-go-tmp TMPDIR=/tmp/hydrogo-go-tmp GOPROXY=off \
  go test ./... -count=1 -p 1 -timeout=8m
GOCACHE=/home/arda/hydrogo/.cache/admin-commit-work/.cache/go-build \
GOTMPDIR=/tmp/hydrogo-go-tmp TMPDIR=/tmp/hydrogo-go-tmp GOPROXY=off \
  go vet ./...
```

Architecture guards also passed:

```sh
go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
go run ./tools/migrationguard -root migrations
go run ./tools/importguard -root tests/external-edition
```

The successful test run includes the root Go packages and their SQLite/unit
contracts. It does not recurse into `tests/external-edition`, which is a nested
module; that module has its own separately passing test and vet run documented
in [`external-edition-generated-admin-2026-10-09.md`](external-edition-generated-admin-2026-10-09.md).
No live PostgreSQL service was configured for this run. Browser E2E assertions
also remain open because this environment denies loopback listener creation;
Playwright discovery alone is not treated as a browser pass.

## Latest regression after mobile editor and fixture-reset changes

After commits `768191be` and `b2ce0cd5`, the complete root Go suite passed
again with:

```sh
GOCACHE=/tmp/hydrogo-go-cache GOMAXPROCS=1 TMPDIR=/var/tmp/hydro-go-tests \
  go test ./... -count=1 -p 1 -timeout=8m
GOCACHE=/tmp/hydrogo-go-cache GOMAXPROCS=1 TMPDIR=/var/tmp/hydro-go-tests \
  go vet ./...
GOCACHE=/tmp/hydrogo-go-cache GOMAXPROCS=1 TMPDIR=/var/tmp/hydro-go-tests \
  go run ./tools/moduleguard -root modules -allow modules/api \
    -dependencies modules/dependencies.json
GOCACHE=/tmp/hydrogo-go-cache GOMAXPROCS=1 TMPDIR=/var/tmp/hydro-go-tests \
  go run ./tools/migrationguard -root migrations
```

The four Edition import guards also passed: `examples/client-edition`,
`examples/news-edition`, `examples/construction-edition`, and
`tests/external-edition`. Migrationguard checked 110 migrations. The full
root suite includes the mobile CSS/CSP contract and generated-richtext fixture
package compile. `HYDRO_TEST_POSTGRES_URL` was unset, so database-specific
PostgreSQL Go contracts were skipped by their existing opt-in gates; the
separate PostgreSQL browser fixture replay evidence remains in
[`generated-richtext-fixture-repeatability-2026-10-09.md`](generated-richtext-fixture-repeatability-2026-10-09.md).
The suite does not recurse into the independent nested Edition Go module.


## Latest PostgreSQL-enabled root regression

The complete root suite was rerun against a disposable PostgreSQL 18.6 cluster
with `HYDRO_TEST_POSTGRES_URL` set:

```sh
GOCACHE=/tmp/hydrogo-go-cache GOMAXPROCS=1 TMPDIR=/var/tmp/hydro-go-tests \
  go test ./... -count=1 -p 1 -timeout=8m
```

All root packages passed, including PostgreSQL-gated contract and backup tests.
The fixture used a temporary `hydro_app` login role with `NOBYPASSRLS` and
`CREATEDB`, and set `HYDRO_PG_PRIVILEGED_ROLE=hydro_app` so backup/restore tests
used the configured test role. The disposable cluster was removed after the
run. This validates root-module tests against local PostgreSQL 18.6; it does
not cover the nested Edition module, production deployment, or replace the
separate authenticated browser runs.

## Regression after Site PostgreSQL RLS correction

On 2026-10-09, the full root suite passed again on SQLite after commit
`ce9c933b`, including the shared SQLite/PostgreSQL Site appearance-route
contract and the new forward migration package:

```sh
TMPDIR=/tmp/hydro-go-tests go test ./... -count=1 -p 1 -timeout=8m
go vet ./...
go run ./tools/migrationguard -root migrations
go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
go run ./tools/importguard -root examples/client-edition
go run ./tools/importguard -root examples/news-edition
go run ./tools/importguard -root examples/construction-edition
go run ./tools/importguard -root tests/external-edition
```

All commands passed; migrationguard checked 111 migrations. A full-root rerun
with `HYDRO_TEST_POSTGRES_URL` was attempted in the current managed execution
environment, but PostgreSQL Unix-socket access failed with `operation not
permitted` before PostgreSQL contracts could run. That is an environment
access limitation, not a PostgreSQL pass. The focused `modules/site/...`
PostgreSQL suite passed against PostgreSQL 18.6 in the prior unrestricted
execution context; this rerun does not recertify every root PostgreSQL
contract after the Site migration. A network lookup for `github.com` is also
blocked in the current environment, so this validation-only follow-up cannot
be pushed from this session.
