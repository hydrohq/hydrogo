# Admin Studio full Chromium acceptance run

**Date:** 2026-10-08  
**Result:** 67/67 Chromium tests passed after repairing generated Admin test
session recovery and keeping MFA screenshots in the isolated test directory.

## Latest full rerun

The full suite was rerun on the current `main` revision with a fresh isolated
SQLite Edition, demo seed, temporary administrator, and generated storage
state. It passed **67/67** in 5.4 minutes. The earlier attempt without this
fixture was stopped after its expected connection-refused failures against
`127.0.0.1:8090`; no product failure was inferred from that invalid setup.

The full Go package suite then passed with temporary directories outside the
repository: `TMPDIR=/var/tmp/hydro-go-tests go test ./... -count=1 -p 1
-timeout=8m`. An initial run with `TMPDIR` inside the repository caused
generator test fixtures to resolve temporary paths as Go import paths; the
generator package and full suite both pass with the standard external temp
directory. `go vet ./...`, actionguard, moduleguard, migrationguard (110
migrations), assetguard, and all three Edition importguards passed as well.

```sh
export HYDRO_E2E_BASE_URL=http://127.0.0.1:8090
export HYDRO_E2E_STORAGE_STATE="$TMPDIR/full-chromium-fixture/auth-state.json"
export HYDRO_E2E_ADMIN_EMAIL=admin-e2e@example.test
export HYDRO_E2E_ADMIN_PASSWORD='<temporary test password>'
TMPDIR=/home/arda/hydrogo/.tmp/pw-system-tmp npx playwright test \
  --project=chromium --workers=1 --reporter=line --trace=off \
  --output=/home/arda/hydrogo/.tmp/chromium-full-fixture
```

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

- Previous full Chromium suite: 66/66 passed, one worker, 5.5 minutes. The
  latest rerun above includes the additional current test and passed 67/67.
- Generated E2E generator package: `go test ./tools/e2egen -count=1` passed.
- Full Go package suite: `TMPDIR=/var/tmp/hydro-go-tests go test ./... -count=1 -p 1 -timeout=5m` passed.
- `go vet ./...` passed.
- `moduleguard`, `migrationguard` (110 migrations), and `assetguard` passed.

The suite covers responsive and accessibility checks, Admin role boundaries,
MFA and password recovery, dynamic Collections CRUD, generated Edition CRUD,
relations/media, editorial workflow, public frontend journeys, and the
metadata-generated Admin route smoke tests. SQLite-backed browser fixtures
were used; this run does not replace PostgreSQL storage contract validation.

## Opt-in locale and visual projects

With `HYDRO_VISUAL=1`, locale and visual acceptance was run against an isolated
SQLite demo fixture:

- Locale/direction: 3/3 passed (Turkish LTR, Arabic RTL bounds, mixed-direction
  text bounds).
- Desktop visual baselines: 7/7 passed (public home/news and five Admin
  surfaces).
- Tablet and mobile visual projects ran in the official Playwright 1.63.0 Noble
  container because the host lacks WebKit system libraries. All 14 WebKit
  screenshots passed after updating three stale goldens: the mobile home golden
  was blank, and tablet/mobile dashboard goldens contained a prior-session card
  absent from the clean fixture. The rendered screens were inspected before
  accepting those baseline updates. Overall locale and visual result: 24/24
  passed (locale 3/3, desktop 7/7, tablet 7/7, mobile 7/7).
- Chromium responsive viewport and Admin route fit checks also passed for
  390px mobile and 768px tablet.

The opt-in run command was:

```sh
HYDRO_VISUAL=1 npm test -- --project=locale --project=visual-desktop \
  --project=visual-tablet --project=visual-mobile --workers=1
```
