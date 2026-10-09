# Independent Edition generated Admin contract — 2026-10-09

The separate `tests/external-edition` Go module was revalidated after the
generated Admin validation and route-test changes.

```sh
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go test ./... -count=1 -p 1 -timeout=5m
GOCACHE="$PWD/.cache/go-build" GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go vet ./...
```

Both commands passed. The contract tests cover:

- explicit generated Admin selection and 404 for an inactive resource;
- the generated SQLite migration, tenant-resolver preflight, and absence of
  partially registered routes when that resolver is missing;
- tenant-scoped create/read and Admin list isolation between two tenants;
- generated Admin create/update/delete, missing-CSRF rejection, viewer read
  and mutation denials, and persisted-state checks after denied mutations.

The Edition is a separate module and imports the public `extension/v1` Admin
contract. These tests use an in-memory HTTP mux and fixture-provided identity,
authorization, tenant, and CSRF callbacks. They do not certify production
session middleware, real browser behavior, PostgreSQL, or deployment setup.
The callback boundary remains Edition-owned as required by ADR-0349/0354/0355.
