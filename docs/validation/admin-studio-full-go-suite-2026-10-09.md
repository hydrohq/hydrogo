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
