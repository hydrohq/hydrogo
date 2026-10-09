# Independent generated Edition role overrides — 2026-10-10

Extended the separate `tests/external-edition` composition contract through the
public generated API and Admin aggregate helpers. The Edition fixture carries
explicit permission overrides through its request policy boundary:

- A Viewer with an explicit create grant can submit the Admin create form;
  the generated API accepts the matching policy decision and SQLite readback
  confirms the row was persisted.
- An Editor update-deny override and an Admin delete-deny override return 403
  from the Admin route and leave the existing SQLite row unchanged.

Validation passed:

```sh
cd tests/external-edition
GOCACHE=/tmp/hydrogo-adminstudio-gocache GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go test ./... -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydrogo-adminstudio-gocache GOTMPDIR=/tmp TMPDIR=/tmp GOPROXY=off \
  go vet ./...
cd ../..
GOCACHE=/tmp/hydrogo-adminstudio-gocache go run ./tools/importguard -root tests/external-edition
```

This exercises Edition-provided role and override callbacks with real generated
SQLite persistence, but test identity is still fixture-provided. It does not
prove deployed auth/session integration or PostgreSQL-specific execution for
these override cases. Those remain open acceptance gates.
