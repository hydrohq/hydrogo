# Admin Studio full Chromium acceptance run

**Date:** 2026-10-08  
**Result:** 66/66 Chromium tests passed after repairing generated Admin test
session recovery and keeping MFA screenshots in the isolated test directory.

The full browser suite needs a demo Edition, seeded content, an Admin account,
and a Playwright storage state. Running it without that fixture produced
connection-refused failures in the tests that target `127.0.0.1:8090`; that
was a runner setup issue, not an application failure.

With the demo Edition running, 58 tests passed and eight generated Admin
surface tests were redirected to login because the saved storage state was no
longer accepted. The generated E2E output now opens `/admin` before each Admin
surface test and re-authenticates with the configured E2E Admin credentials
when it reaches login. The MFA browser test also writes its visual artifact to
its isolated temporary directory instead of a repository path that may be
owned by another user.

Validation completed:

- Full Chromium suite: 66/66 passed, one worker, 4.3 minutes.
- Generated E2E generator package: `go test ./tools/e2egen -count=1` passed.
- Full Go package suite: `TMPDIR=/var/tmp/hydro-go-tests go test ./... -count=1 -p 1 -timeout=5m` passed.
- `go vet ./...` passed.
- `moduleguard`, `migrationguard` (110 migrations), and `assetguard` passed.

The suite covers responsive and accessibility checks, Admin role boundaries,
MFA and password recovery, dynamic Collections CRUD, generated Edition CRUD,
relations/media, editorial workflow, public frontend journeys, and the
metadata-generated Admin route smoke tests. SQLite-backed browser fixtures
were used; this run does not replace PostgreSQL storage contract validation.
