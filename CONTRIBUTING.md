# Contributing to HydroGo

HydroGo is developed around a small, portable Core with explicit module and
Client Edition boundaries. Contributions should improve a real customer or
operator outcome without adding infrastructure that an active profile does not
need.

## Before making a change

Read [AGENTS.md](AGENTS.md), the [philosophy](docs/HYDROGO_PHILOSOPHY.md),
the [implementation plan](docs/implementation_plan.md), the
[architecture index](docs/architecture/README.md), and every affected ADR.
Choose one concrete acceptance criterion from the [roadmap](docs/ROADMAP.md).
Do not silently change an accepted architectural decision; supersede it with a
new ADR and link the two decisions.

## Development rules

1. Use a short-lived branch and target one concrete outcome per pull request.
2. Keep the SDK at v0 until two independent modules or Editions demonstrate a
   stable general contract.
3. Add the narrowest useful test first. Changes to database behavior must cover
   both SQLite and PostgreSQL contracts when the behavior is shared.
4. If a migration is needed, include dialect handling, rollback or compatibility
   metadata, and the corresponding contract tests. Never rewrite a published
   migration.
5. Keep SQL dialect, cache-driver, and deployment-proxy details out of domain
   and application services.
6. Disabled modules must not create routes, migrations, workers, assets, or
   mandatory external services.
7. Treat paths, uploaded data, manifests, restore markers, and operator input as
   untrusted. Preserve bounded reads, symlink defenses, idempotency, and
   fail-closed behavior.
8. Record capacity, deployment, compatibility, security, and data-lifecycle
   effects in the pull request when they apply.

## Local verification

The supported toolchain is Go 1.27.0. Run the focused tests while iterating and
the complete local gate before opening a pull request:

```powershell
go test ./...
go vet ./...
powershell -ExecutionPolicy Bypass -File .\scripts\verify-local.ps1
```

For PostgreSQL and production-style checks, configure an isolated
`HYDRO_TEST_POSTGRES_URL` and run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\verify-production-gates.ps1
```

The equivalent shell script is `scripts/verify-production-gates.sh`. Linux CI
is authoritative for the full race job when a local Windows environment has no
C compiler. Never report a command as passing unless it actually ran and
passed. A PostgreSQL contract test must not be silently treated as skipped.

## Pull requests and commits

Pull requests should explain the problem, the intended behavior, the tests
executed, and any remaining evidence gap. Prefer focused Conventional Commit
messages such as `fix(cache): reject stale refill` or `docs(roadmap): record
restore evidence`.

Do not commit secrets, customer data, database files, backups, personal media,
or generated build output. Keep the working tree clean after verification and
push only the commits that belong to the change.
