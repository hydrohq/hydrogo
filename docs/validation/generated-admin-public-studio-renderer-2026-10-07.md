# Public Generated Admin Studio Shell Validation

**Date:** 2026-10-07  
**Scope:** Public Edition composition, shared shell, generated navigation,
locale, and Chromium CRUD flow.

## Result

An external Edition can render generated Admin pages with the shared HydroGo
Studio shell using only the versioned `extension/v1/adminstudio` package. The
example Edition no longer imports Core's `internal/ui` package. The public
renderer receives an Edition-resolved identity for shell presentation; route
authorization and per-resource visibility still come from required Edition
callbacks.

The actual Project Edition server ran in Chromium. The generated collection
appeared in the shared sidebar, CRUD/search/filter flows passed, the first-run
Admin guide could be dismissed, and the generated list/edit screens passed
axe-core WCAG 2.1 A/AA checks. At 390px and 768px widths the page had no
document-level horizontal overflow. The page language cookie selected both
Turkish and English labels consistently between the shared shell and generated
navigation. The desktop capture was inspected after the language consistency
fix.

## Commands

```sh
go test ./internal/adminstudio ./extension/v1/adminstudio ./examples/project-edition/... -count=1
go vet ./extension/v1/adminstudio ./internal/adminstudio ./examples/project-edition/...
cd tests/external-edition && go test -mod=mod ./...
cd e2e && npx playwright test custom/generated-admin-crud.spec.ts --project=chromium --workers=1 --reporter=line
go test -mod=mod ./...
```

All listed checks passed. The detector ran once on the changed generated Admin
runtime source and reported no findings. Browser captures and traces remain in
the ignored `e2e/artifacts/` directory.

This validates composition and presentation for the shared generated Admin
runtime. Resource activation, authorization, CSRF, tenant scope, migrations,
and editorial callbacks remain explicit Edition responsibilities. The full
Blueprint-to-first-CRUD manual-step and timing benchmark remains open.
