# Metadata-driven E2E

`e2e/manifest.json` is an explicit contract for both Admin Studio and the
generated frontend. It is intentionally not inferred from arbitrary HTML.

Generate deterministic Playwright tests with:

```powershell
go run ./tools/e2egen -manifest e2e/manifest.json -output e2e/generated/hydro.spec.ts
```

After installing the test-only dependency, run them with:

```powershell
npm install
npx playwright install chromium
npm run generate
$env:HYDRO_E2E_STORAGE_STATE = "auth-state.json"
npm run setup-auth
npm test

# Approved desktop visual goldens (requires a seeded server)
HYDRO_VISUAL=1 npx playwright test visual.spec.ts
# Create/update goldens only during a reviewed baseline change:
HYDRO_VISUAL=1 npx playwright test visual.spec.ts --update-snapshots
# Locale/direction and long mixed-direction text fixtures
HYDRO_VISUAL=1 npx playwright test locale.spec.ts
```

The storage state is optional for public frontend tests and required for the
generated protected Admin Studio tests. Create it only in an isolated local or
CI environment; never use a production session file.

The generated file is disposable. Customer-specific journeys belong under
`e2e/custom/` and are never overwritten by the generator. The application must
provide the declared `data-testid` values; unknown routes or missing metadata
must fail generation or the browser test.

The local orchestration also builds `packages/client` and runs its
typed-client smoke journey against the seeded versioned API.
