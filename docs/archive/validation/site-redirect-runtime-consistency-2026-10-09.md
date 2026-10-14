# Site redirect persistence and runtime consistency — 2026-10-09

## Reproduced failure

With `/old → /new` already registered, the real Admin route accepted
`/new → /old` as HTTP 204 and persisted it before the in-memory registry
rejected the cycle. The regression test first failed with the persisted row
still present after rejection.

## Change

`RedirectRegistry` now serializes snapshot writers, compiles and validates the
candidate snapshot before persistence, runs the durable write, and only then
publishes the validated snapshot. Callback failure leaves the previous snapshot
intact. Admin legacy redirect save, rule create/import/delete and 404 conversion
use this path, so database failures or detected cycles cannot produce a
successful response with an unapplied in-memory update. Redirect inputs are
normalized consistently before both storage and runtime registration; unsafe
protocol-relative targets remain rejected.

Tests also verify that a wildcard rule becomes live immediately, deletion
removes it from the runtime matcher, and a cyclic CSV batch writes no rules.
The registry unit test checks that a failed persistence callback preserves the
old snapshot and that a cycle is rejected before the persistence callback.

## Verification

```sh
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go test ./modules/site/... -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go test ./... -count=1 -p 1 -timeout=8m
GOCACHE=/tmp/hydrogo-go-build GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go vet ./...
GOCACHE=/tmp/hydrogo-go-race GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go test -race ./modules/site -run '^TestRedirectRegistry' -count=1 -p 1
GOCACHE=/tmp/hydrogo-go-race GOMAXPROCS=1 GOPROXY=off TMPDIR=/tmp/hydrogo-admin-tests \
  go test -race ./modules/site/transport -run 'TestSite(RedirectAdminRoutePersistsOnlyAuthorizedCSRFMutation|RedirectRuleAdminRoutesPersistAuthorizedCRUD|RedirectRuleCSVImportIsAuthorizedAndAllOrNothing)$' -count=1 -p 1
```

All listed tests and vet passed. moduleguard, migrationguard (110 migrations),
external Edition tests/vet/importguard also passed. `go test ./...` does not
enable PostgreSQL without `HYDRO_TEST_POSTGRES_URL`; no PostgreSQL claim is made.
Registry snapshots are process-local as documented by ADR-0268, so invalidation
across multiple live instances remains outside this change.
