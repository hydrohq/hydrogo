# Generated block editor theme matrix — 2026-10-09

## Change

The generated Editor.js form browser journey now checks the real editor in all
four Admin theme families (`hydrogo`, `shadcn`, `nova`, `aurora`) and both light
and dark modes. Each variant asserts the active theme attributes, a mounted
editable block, and axe WCAG 2.1 A/AA results before the CRUD round trip.

## Verification status

- The latest generated rich-text Chromium suite passed **10/10** in 1.1 min using the
  externally started local fixture. It includes Editor.js light/dark coverage
  for all four theme families, mounted editor assertions, and axe WCAG 2.1
  A/AA checks before CRUD round-trip assertions. Its theme accessibility test
  checks five generated/editorial screens across four theme families and two
  color modes (40 screen/theme cells).
- The first in-process startup attempt was blocked by a loopback bind
  restriction. Starting the fixture separately and using its loopback URL let
  Playwright run without owning the listening socket; no product code was
  changed to work around the runner.
- Exact successful command:

  ```sh
# Terminal 1, from the repository root:
  HYDRO_E2E_ADDRESS=127.0.0.1:8099 GOCACHE=/tmp/hydrogo-go-cache \
    go run ./tests/e2e/support/generatedrichtext
# Terminal 2, from e2e:
cd e2e
  HYDRO_E2E_EXTERNAL_SERVER=1 HYDRO_E2E_BASE_URL=http://127.0.0.1:8099 \
    TMPDIR=/var/tmp/hydro-go-tests npx playwright test \
    custom/generated-richtext.spec.ts --project=chromium --workers=1 \
    --reporter=line --trace=off
  ```

## Open gate

The theme matrix is verified against a local synthetic generated-richtext
fixture. It is not a production Edition deployment or customer identity test;
those wider gates remain open. The external-fixture path below is available for
a local browser runner that cannot bind sockets itself.

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

The external fixture is a local test server with synthetic identity/data, not
customer deployment evidence.
