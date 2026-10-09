# Generated rich-text fixture repeatability — 2026-10-09

## Root cause and fix

An external-fixture replay retained data from the previous browser suite. The
next run hit duplicate fixed article titles, and the locale/draft flows saw
mutated seed records. The fixture now exposes `POST /__test/reset`; the E2E
suite calls it after its health check. Reset clears in-memory state and, when
using PostgreSQL, transactionally truncates and reseeds the fixture tables.
This endpoint exists only in `tests/e2e/support/generatedrichtext`.

## Verification

The complete Chromium suite passed twice without restarting the fixture:

| Store | Replay 1 | Replay 2 |
| --- | ---: | ---: |
| In-memory fixture | 8/8, 29.6 s | 8/8, 29.8 s |
| PostgreSQL 18.6 fixture schema | 8/8, 32.5 s | 8/8, 29.6 s |

Both suites covered generated rich-text and block CRUD, media/relation,
validation/retry, conflict recovery, draft review/discard, locale publishing,
axe scans, and the mobile editor layout bound. The PostgreSQL instance was a
temporary local cluster, and the fixture used its own temporary schema; the
cluster and fixture were stopped and removed after the runs.

The browser command was run twice against each already-running fixture:

```sh
cd e2e
HYDRO_E2E_EXTERNAL_SERVER=1 \
  HYDRO_E2E_BASE_URL=http://127.0.0.1:<fixture-port> \
  TMPDIR=/var/tmp/hydro-go-tests \
  npm exec playwright test custom/generated-richtext.spec.ts -- \
  --project=chromium --workers=1 --reporter=line --trace=off
```

The suite calls the reset endpoint on each invocation. The PostgreSQL fixture
is synthetic and schema-isolated; these runs do not represent production
Edition migrations, authentication, tenant isolation, or deployment.

`go test ./tests/e2e/support/generatedrichtext ./internal/ui ./internal/adminstudio -count=1`,
`go vet ./tests/e2e/support/generatedrichtext ./internal/ui ./internal/adminstudio`,
and `git diff --check` passed.
