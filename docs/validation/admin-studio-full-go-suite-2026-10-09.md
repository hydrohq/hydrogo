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
