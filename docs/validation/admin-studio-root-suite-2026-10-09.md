# Admin Studio root-suite verification — 2026-10-09

This checkout passed the complete root Go suite twice after the latest Admin
Studio route and font-localization test changes:

```sh
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go test ./... -count=1 -p 1 -timeout=8m
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go test ./... -shuffle=on -count=1 -p 1 -timeout=8m
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go vet ./...
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go run ./tools/migrationguard -root migrations
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go run ./tools/importguard -root examples/client-edition
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go run ./tools/importguard -root examples/news-edition
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go run ./tools/importguard -root examples/construction-edition
```

Both `go test` commands and `go vet` exited successfully. `moduleguard` reported
OK, `migrationguard` reported 110 migrations, and all three Edition import
guards exited successfully. These commands validate the repository suite and
static architecture boundaries in this checkout; they do not imply PostgreSQL
or browser certification.

## PostgreSQL and browser gates

`HYDRO_TEST_POSTGRES_URL` is unset. PostgreSQL 18.6 `initdb` successfully
created a disposable cluster under `/tmp`, but `pg_ctl` could not bind the
Unix-domain socket (`Operation not permitted`). The sandbox also denied Docker
API access. PostgreSQL tests were therefore not enabled by this run.

The Playwright dependencies are available from the local npm cache, and
`npm ci --offline` plus `playwright --list` succeeded. The independent Edition
SQLite fixture server also completed migration and route composition but could
not bind `127.0.0.1:8099` (`socket: operation not permitted`). Browser
assertions remain unexecuted. These external execution gates must run on a
runner that permits a PostgreSQL socket and loopback listener.
