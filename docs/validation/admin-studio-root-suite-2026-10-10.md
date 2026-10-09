# Admin Studio root suite — 2026-10-10

After the generated widget interface-localization change, the complete root Go
test suite and vet ran against the committed tree. This validates the shared
typed components and generated Admin handler against the rest of the workspace.

Passed:

```sh
GOCACHE=/tmp/hydro-admin-gocache go test ./... -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydro-admin-gocache go vet ./...
(cd tests/external-edition && GOCACHE=/tmp/hydro-admin-gocache go test ./...)
GOCACHE=/tmp/hydro-admin-gocache go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
GOCACHE=/tmp/hydro-admin-gocache go run ./tools/migrationguard -root migrations
GOCACHE=/tmp/hydro-admin-gocache go run ./tools/importguard -root tests/external-edition
```

The external Edition module passed independently; moduleguard and external
importguard passed; migrationguard reported 111 migrations. This was the
workspace SQLite-backed Go suite. It does not certify a live PostgreSQL server,
browser journey, deployed Edition auth/session composition, or production
release configuration. Those acceptance gates remain open in the Admin Studio
audit plan.
