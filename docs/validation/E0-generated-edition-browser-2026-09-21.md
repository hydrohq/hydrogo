# Generated Edition browser contract — 2026-09-21

`scripts/verify-generated-editions.ps1` runs compatibility, generated Go
migration and composition-root HTTP tests, generated TypeScript SDK compilation, the isolated Core
`hydro migrate`/`hydro serve` readiness smoke, and Playwright together.
Playwright renders the checked-in
Project and Article generated table/form
artifacts in Chromium and verifies visible table/form structure, required
inputs, and the absence of executable script tags:

```text
npx playwright test custom/generated-editions.spec.ts --reporter=line
2 passed
```

This is a generated-artifact browser contract. It does not claim a full
production server journey, authentication topology, or customer Edition
deployment; those remain separate staging gates.
