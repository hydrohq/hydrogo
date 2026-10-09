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
- `hydro new resource` now makes the first string or text field the default Admin title,
  list column, and search field. The wizard test checks both the authored
  Blueprint and generated `admin_manifest.json`; resources without a string or text
  field still require an explicit title/list choice.
- The Project Edition's `milestone.blueprint.yaml` was authored through
  `hydro new resource` and generated alongside its migration proposal, typed
  API, Admin metadata, and test contract. Because the Edition already invokes
  `RegisterGeneratedAdmin`, the new resource appeared in the shared Admin
  registry without a second ID edit. The Edition explicitly added
  `RegisterMilestoneRoute` with its own policy and applied the reviewed
  migration at startup; those are deliberate human-owned activation steps.
- The Project Edition HTTP integration test proves SQLite migration, Admin
  navigation, API create, and Admin list readback. Its opt-in PostgreSQL test
  applies the same generated migration in an isolated schema, checks Admin
  navigation, creates through the API, and reads the record in the Admin list.
  That test passed on an isolated PostgreSQL 18.6 cluster.
- The generated resource's Chromium flow now creates, reads back from the API,
  edits, and deletes a milestone through the shared Admin. Axe checks run on
  its list and form, and the generated form is checked at 390px without
  horizontal overflow.
- The new migration is additive and declares SQLite/PostgreSQL expand
  compatibility, checksums, and forward/restore-only recovery in
  `generated/milestone/migration.yaml`. It adds one table and no worker, cache,
  pool, or separate asset. The Project Edition example is explicitly a
  loopback-only demo with permissive policy; it is not production auth.
- The new axe pass found a 3.7:1 collection-description contrast on the
  existing shared Admin resource page. `.dashboard-card-head .muted` now uses
  `--ink-secondary`; the 8 theme/mode generated-list axe checks passed again.
  The shared red alert's thick left accent (including its shadcn override) was
  also replaced with a neutral one-pixel border; alert color and surface remain
  semantic red tokens.

## Commands

```sh
go test ./internal/generator ./examples/project-edition/... ./extension/v1/adminstudio -count=1
go test ./cmd/hydro -count=1
go test ./examples/project-edition/... -count=1
go test ./examples/project-edition/custom -run '^TestPostgresEditionGeneratedAdminCRUD$' -count=1
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

For the PostgreSQL test, set `HYDRO_TEST_POSTGRES_URL` to a disposable test
database URL before running the command. The recorded 2026-10-07 run used an
isolated local PostgreSQL 18.6 cluster and schema.

This removes manual per-resource activation-list maintenance after Edition
composition has been wired. It does not remove the deliberate one-time work of
connecting the Edition API, policies, CSRF and shell identity, nor migration
review, server startup, or first-browser-run validation.

## Generated record edit theme accessibility follow-up — 2026-10-08

The native generated Project Edition's real record-edit screen now receives an
axe-core WCAG 2.1 A/AA scan under HydroGo, shadcn, Nova, and Aurora themes in
both light and dark modes. All eight variants passed as part of the full
generated CRUD browser journey (create, search/filter, edit, delete): 1/1
Chromium test. The list and generated create form theme checks remain covered
by the same journey.

## Runtime wiring authorization guidance — 2026-10-09

Generated `RUNTIME_WIRING.md` now tells Edition authors to derive optional
`ResourceAccess` from the same user, tenant, and record rules as the API. It
documents that the callback governs UI affordances and direct Admin routes,
while API authorization remains the boundary for all callers. A generator test
asserts this guidance is emitted. This is setup documentation and does not
replace authorization contract tests.

The checklist now makes the safe adoption path explicit: new Edition
compositions should always bind `ResourceAccess` before exposing the Admin.
Omitting the optional v1 callback is identified as legacy compatibility that
shows all basic CRUD actions; the API remains the authorization boundary. The
generated Project Edition article guide and its artifact lock were regenerated.
`TestGeneratedRuntimeWiringExplainsEditionCRUDCapabilities`, all
`internal/generator` and `cmd/hydro` tests, their vet checks, moduleguard,
migrationguard, and `git diff --check` passed.
