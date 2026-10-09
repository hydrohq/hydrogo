# Admin Studio root suite — 2026-10-10

After the generated and legacy dynamic Admin locale-control changes, the
complete root Go test suite and vet ran against the committed tree. The
independent external Edition module was also retested. This validates the
shared Admin shell, typed components, generated Admin handler, and fixture
module against the rest of the workspace.

Passed:

```sh
GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./... -count=1 -p 1 -timeout=5m
GOCACHE=/tmp/hydrogo-adminstudio-gocache go vet ./...
(cd tests/external-edition && GOCACHE=/tmp/hydrogo-adminstudio-gocache go test ./... -count=1 -p 1 -timeout=5m && GOCACHE=/tmp/hydrogo-adminstudio-gocache go vet ./...)
GOCACHE=/tmp/hydrogo-adminstudio-gocache go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
GOCACHE=/tmp/hydrogo-adminstudio-gocache go run ./tools/migrationguard -root migrations
GOCACHE=/tmp/hydrogo-adminstudio-gocache go run ./tools/importguard -root examples/client-edition
GOCACHE=/tmp/hydrogo-adminstudio-gocache go run ./tools/importguard -root examples/news-edition
GOCACHE=/tmp/hydrogo-adminstudio-gocache go run ./tools/importguard -root examples/construction-edition
GOCACHE=/tmp/hydrogo-adminstudio-gocache go run ./tools/importguard -root tests/external-edition
```

The external Edition module passed independently; moduleguard and all four
importguard roots passed; migrationguard reported 111 migrations. This was the
workspace SQLite-backed Go suite. A focused Chromium run was also retried with
the writable Go cache: migration completed, then the test fixture failed to
bind `127.0.0.1` with `listen EPERM`, before any browser assertions. It does not
certify a live PostgreSQL server, browser journey, deployed Edition auth/session
composition, or production release configuration. Those acceptance gates
remain open in the Admin Studio audit plan.
