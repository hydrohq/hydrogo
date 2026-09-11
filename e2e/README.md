# Metadata-driven E2E

`e2e/manifest.json` is an explicit contract for both Admin Studio and the
generated frontend. It is intentionally not inferred from arbitrary HTML.

Generate deterministic Playwright tests with:

```powershell
go run ./tools/e2egen -manifest e2e/manifest.json -output e2e/generated/hydro.spec.ts
```

The generated file is disposable. Customer-specific journeys belong under
`e2e/custom/` and are never overwritten by the generator. The application must
provide the declared `data-testid` values; unknown routes or missing metadata
must fail generation or the browser test.
