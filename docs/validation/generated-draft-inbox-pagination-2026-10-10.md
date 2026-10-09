# Generated draft inbox pagination — 2026-10-10

Implemented ADR-0367. Draft-enabled Editions may provide `ListDraftPage` with
explicit offset and limit; the Admin requests 25 rows plus one lookahead,
validates pages 1–100, trims the lookahead, and renders localized GET
Previous/Next controls. Search, filter, and sort controls are not presented
because draft storage has no such query contract. Legacy `ListDrafts` callbacks
keep their first-page behavior and do not receive misleading page controls;
page > 1 is rejected before storage if no offset-aware callback is composed.

Validation passed:

```sh
GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./internal/adminstudio ./extension/v1/adminstudio -count=1
GOCACHE=/tmp/hydrogo-adminstudio-gocache go vet ./internal/adminstudio ./extension/v1/adminstudio
cd tests/external-edition
GOCACHE=/tmp/hydrogo-adminstudio-gocache GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go test ./... -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydrogo-adminstudio-gocache GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off go vet ./...
cd ../..
GOCACHE=/tmp/hydrogo-adminstudio-gocache go run ./tools/importguard -root tests/external-edition
cd e2e && ./node_modules/.bin/playwright test custom/generated-richtext.spec.ts --list
```

The root `go test ./... -count=1 -p 1 -timeout=8m` run completed with a single
environmental failure: `cmd/hydro` could not link its temporary binary because
the `/tmp` tmpfs was full. The failing `TestRunEditionDevServesHealthzAndReactsToThemeChanges`
passed when rerun with the workspace disk-backed Go temp directory. Generator
tests initially also inherited that workspace temp directory and treated its
path as a module import; the four affected generator tests passed rerun with
`TMPDIR=/tmp`. Full root `go vet ./...` passed. Each affected package was
verified independently, although the whole root suite did not pass in one
invocation under the runner's temporary-storage limit.

The external Edition contract test applies a real SQLite table and verifies
stable order, 25-row boundaries, page-two readback, and localized pager
rendering through the public generated Admin contract. The browser test also
asserts page size, keyboard activation, locale preservation, and axe results;
it was discovered but runtime startup was denied by `listen EPERM` binding
`127.0.0.1`, so browser/axe assertions remain unverified. The optional
PostgreSQL fixture now uses stable `updated_at, resource_id` ordering and
offset/limit SQL, but PostgreSQL execution of this path remains open because no
test URL/server is available in this environment.
