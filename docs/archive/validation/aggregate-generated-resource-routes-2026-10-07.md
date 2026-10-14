# Blueprint Aggregate API Route Registration Validation

**Date:** 2026-10-07  
**Status:** SQLite and PostgreSQL 18.6 Project Edition flows passed.  
**Decision:** [ADR-0355](../architecture/0355-aggregate-generated-resource-route-registration.md)

## Validated Flow

- The Blueprint set within a Go module generates a deterministic
  `generated.RegisterGeneratedResources`. For each resource, the helper checks
  the Edition policy callback, and for tenant-scoped resources, checks the resolver requirement;
  when a dependency is missing, no routes are registered.
- `examples/project-edition` wires two generated resources with a single aggregate API
  call. SQLite integration tests verify API create/read, Admin list
  views, and that all routes remain disabled if a resource policy is missing.
- In the independent `tests/external-edition` Go module, a tenant-scoped
  `article` is generated via Blueprint. Generated migrations are applied to SQLite; API create,
  same-tenant read, cross-tenant 404, generated Admin create/update/delete,
  CSRF rejection, viewer deny, and separate per-tenant Admin lists are verified
  in authentic HTTP tests against the same SQLite database. Missing tenant resolvers
  are rejected at startup and prevent route registration.
- The same independent Edition is started with SQLite on a separate Chromium E2E server;
  record create/edit/delete via browser Admin, viewer deny, cross-tenant
  API 404, and record absence in the other tenant's list are verified.
- `hydro project check` verifies the aggregate helper against the module path and Blueprint set;
  generator tests reject manual tampering with the helper.
- The relative `--out tests/external-edition` path previously caused the same directory
  to be appended twice during the CLI scaffold `gofmt` step. The CLI now converts the verification
  root to an absolute path; an additional regression test runs gofmt and `go test ./...`
  at the relative root. Full scaffold verification with a relative `--out` also passed.

## Results

- `go test ./... -count=1 -p 1 -timeout=5m` passed.
- `go vet ./...`, actionguard, moduleguard, migrationguard (110 migrations), and
  Client Edition importguards passed. The new independent
  `tests/external-edition` fixture also passed the `importguard` check.
- `go test ./internal/generator ./examples/project-edition/... -count=1`,
  external Edition `go test -mod=mod ./... -count=1`, relative scaffold CLI
  regression, and `hydro project check` stale-helper test passed.
- Generated Admin Chromium CRUD passed **1/1**.
- Independent Edition Chromium Admin CRUD and tenant/role flow passed **1/1**.
- On an isolated PostgreSQL 18.6 cluster,
  `HYDRO_TEST_POSTGRES_URL=... go test ./examples/project-edition/custom -run '^TestPostgresEditionGeneratedAdminCRUD$' -count=1` passed **1/1**.
  API create and generated Admin list readback were performed using the same aggregate helper.
- The aggregate helper routes existing resource SQL transport; it introduces no new query,
  schema, transaction, or migration behavior.

## Commands

```sh
go test ./internal/generator ./examples/project-edition/... -count=1
go test ./cmd/hydro -run TestGeneratedScaffoldVerificationResolvesRelativeRoot -count=1
```

The independent Edition package test is executed from the `tests/external-edition` directory
with `go test -mod=mod ./... -count=1`. Full generated Admin Chromium
CRUD test also passed **1/1**:

```sh
cd e2e
npx playwright test custom/generated-admin-crud.spec.ts --project=chromium --workers=1 --reporter=line
```

## Boundaries

When the helper is explicitly called, all aggregate Blueprint resources are wired to the
API; partial Editions directly use the selected `Register<Name>Route` functions.
Auth, policy, CSRF, tenant scope, and migration decisions belong to the Edition.
The generator does not execute DDL. Because `hydro project new` does not yet generate a runnable
Go Edition/server, end-to-end time from a clean scaffold to initial browser CRUD remains open.
`HYDRO_TEST_POSTGRES_URL` was not set in this run; this change does not alter SQL/schema
behavior, and existing SQLite/PostgreSQL resource contracts are preserved.
