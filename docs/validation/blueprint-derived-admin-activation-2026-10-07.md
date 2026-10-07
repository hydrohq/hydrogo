# Blueprint-derived Admin activation validation

**Date:** 2026-10-07  
**Scope:** Keep generated Admin resource selection synchronized with the
Blueprint aggregate, while leaving security and runtime activation under the
Edition composition root.

## Result

The generator now emits `generated/admin_registration.go` with
`RegisterGeneratedAdmin`. That function calls the public Admin Studio contract
with a deterministic list of every resource present in the generated Admin
registry. Adding a Blueprint to an already wired Edition updates its aggregate
registry and registration list together; the Edition does not maintain a
second hand-written list of resource IDs.

The helper does not run at package initialization. An Edition must explicitly
call it from its composition root and supply its API, authorization, resource
visibility, CSRF, and page rendering callbacks. To run a deployment with only
some generated resources, the Edition can use the lower-level
`adminstudio.RegisterGenerated` API with an explicit reviewed selection.
Migration approval and application remain separate.

`hydro project check` verifies the generated registration source matches the
aggregate registry computed from generated specs. Missing, symlinked, edited,
or stale registration output fails validation.

## Evidence

- Generator test creates two resources and asserts the generated helper
  contains both IDs in stable sorted order.
- Generator compatibility test tampers with the helper and confirms the
  project checker rejects it.
- `examples/project-edition/custom/composition.go` calls the generated helper;
  its Go package test compiles and exercises registration.
- Project Edition Chromium generated Admin test passes generated navigation,
  shared shell, create/search/filter/edit/delete, axe checks, and mobile/tablet
  overflow. The generated route is reached through the new helper.
- `hydro new resource` now makes the first string field the default Admin title,
  list column, and search field. The wizard test checks both the authored
  Blueprint and generated `admin_manifest.json`; resources without a string
  field still require an explicit title/list choice.

## Commands

```sh
go test ./internal/generator ./examples/project-edition/... ./extension/v1/adminstudio -count=1
go test ./cmd/hydro -count=1
go vet ./internal/generator ./examples/project-edition/... ./extension/v1/adminstudio
cd examples/project-edition && HYDRO_DATABASE_DSN=file:/tmp/hydro-project-edition-check.sqlite go run ../../cmd/hydro project check
cd e2e && npx playwright test custom/generated-admin-crud.spec.ts --project=chromium --workers=1 --reporter=line
go test -mod=mod ./...
go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
go run ./tools/migrationguard -root migrations
go run ./tools/importguard -root examples/client-edition
go run ./tools/importguard -root examples/news-edition
go run ./tools/importguard -root examples/construction-edition
```

This removes manual per-resource activation-list maintenance after Edition
composition has been wired. It does not remove the deliberate one-time work of
connecting the Edition API, policies, CSRF and shell identity, nor migration
review, server startup, or first-browser-run validation.
