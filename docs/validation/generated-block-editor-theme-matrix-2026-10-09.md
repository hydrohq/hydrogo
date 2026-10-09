# Generated block editor theme matrix — 2026-10-09

## Change

The generated Editor.js form browser journey now checks the real editor in all
four Admin theme families (`hydrogo`, `shadcn`, `nova`, `aurora`) and both light
and dark modes. Each variant asserts the active theme attributes, a mounted
editable block, and axe WCAG 2.1 A/AA results before the CRUD round trip.

## Verification status

- Playwright discovered all seven tests in
  `custom/generated-richtext.spec.ts`.
- The run stopped in suite setup before any test assertion: the generated
  fixture could not bind its loopback server (`listen EPERM: operation not
  permitted 127.0.0.1`). The theme matrix result is therefore **unverified**.
- `git diff --check` passed.

## Open gate

The repository's `browser-e2e` CI job invokes the complete Playwright suite on
Windows, where its isolated loopback fixture can start. Once the branch reaches
CI, inspect the generated-richtext result and attached artifacts before closing
this gate. The external-fixture path below is available for a local browser
runner that cannot bind sockets itself.

The spec supports an externally started local fixture so the Playwright runner
does not need to create a listening socket itself. In one terminal from the
repository root, start the fixture:

```sh
HYDRO_E2E_ADDRESS=127.0.0.1:8099 GOCACHE=/tmp/hydrogo-go-cache \
  go run ./tests/e2e/support/generatedrichtext
```

Then run the browser suite in another terminal:

```sh
cd e2e
HYDRO_E2E_EXTERNAL_SERVER=1 HYDRO_E2E_BASE_URL=http://127.0.0.1:8099 \
  npx playwright test custom/generated-richtext.spec.ts --project=chromium --workers=1
```

Record actual axe findings and screenshots before treating the generated block
editor's theme/accessibility coverage as complete. The external fixture is a
local test server with synthetic identity/data, not customer deployment
evidence.
