# Generated validation link contrast and PostgreSQL CRUD

## Finding

The generated validation summary rendered field links with the browser's
default blue link color on the `.studio-alert` red surface. Chromium axe
reported a WCAG color contrast failure (blue `#0000ee` on `#450a0a`, 1.71:1)
for the `Project code` anchor. This was a product UI defect.

## Change

The generated Admin shell styles alert links to inherit the alert foreground
and show an underline with offset. In light mode, the alert already switches to
`#991b1b`, which maintains contrast on the alert surface. The accessibility
CSS contract test now prevents regression. The native Edition E2E fixture now
supplies the required Project title before the successful create and expects
the Turkish validation copy served by the active locale.

## Verification

- `go test ./internal/ui ./internal/ui/components -count=1 -p 1` — passed.
- `HYDRO_E2E_DATABASE_DIALECT=postgres HYDRO_E2E_DATABASE_DSN='postgres://arda@/hydro_project_native_20261009?host=/run/postgresql&sslmode=disable' npm test -- --project=chromium --workers=1 custom/generated-admin-crud.spec.ts` — passed, 1/1, on PostgreSQL 18.6.
- The browser assertion scans validation feedback with axe across HydroGo, shadcn, Nova, and Aurora in light/dark modes, then performs generated Project create/list/API readback/edit/delete and a Blueprint-generated Milestone validation/create/edit/delete journey.
- `HYDRO_TEST_POSTGRES_URL=... HYDRO_TEST_POSTGRES_DSN=... go test ./... -count=1 -p 1 -timeout=8m` — passed against freshly recreated PostgreSQL 18.6 test databases.
- `go run ./tools/actionguard`, `go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json`, `go run ./tools/migrationguard -root migrations`, and `go run ./tools/importguard -root tests/external-edition` — passed.

## Limits

This verifies one native Project Edition CRUD journey and its generated form
validation surfaces. It does not close the larger Admin Studio product audit,
role/route certification, other browser workflow matrices, or production
release gates.
