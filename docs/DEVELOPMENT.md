# Development

## Run it

```bash
go run ./tools/devserver
```

This rebuilds on save, seeds disposable demo data and serves the API at
http://127.0.0.1:8090 (admin at `/admin`). Use `-address 0.0.0.0:8090` to
open it from a phone on the same network.

Optional local PostgreSQL/Redis: `docker compose -f compose.dev.yaml up -d`.

## Check it

```bash
go run ./tools/check          # or: make check
go run ./tools/check -short   # skip tests
```

To run the GitHub CI jobs themselves on this machine (Docker and [act](https://github.com/nektos/act) needed):

```bash
scripts/ci-local.sh                   # go, first-run, sveltekit-starter, crm-starter
scripts/ci-local.sh go                # one job
```

The script moves the PostgreSQL and Redis service ports away from 5432 and 6379 and skips artifact uploads. The container image build, the Windows jobs and CodeQL cannot run this way.

`check` runs build, vet, gofmt, the repository guards and the test suite. The
pre-push hook runs it; install hooks with `sh scripts/install-git-hooks.sh`.
CI runs more (PostgreSQL/MySQL contracts, browser e2e, CodeQL);
`scripts/verify-local.ps1` reproduces most of it locally and takes about 40
minutes.

### Guards

| Guard | Rule |
|---|---|
| `tools/codeguard` | No HTML in Go string literals; files stay under 800 lines. Existing violations are frozen in `tools/codeguard/baseline.json` and may only shrink. After you remove some, run `go run ./tools/codeguard -update` and commit the smaller baseline. |
| `tools/migrationguard` | Migration metadata, checksums and dependencies are consistent. |
| `tools/moduleguard` | `modules/*` only import what `modules/dependencies.json` declares. |
| `tools/assetguard` | Theme asset cache-busters agree across templates. |

## Generated files

- Theme and admin CSS: edit the `*.source.css` / `input.css` file, then run the
  matching `scripts/build-*-tailwind.ps1` with the standalone Tailwind CLI.

## Migrations

Add a directory `migrations/<YYYYMMDDNNNN>_<name>/` with `migration.yaml`,
`sqlite.sql`, `postgres.sql` and `mysql.sql`. Look at a recent one for the
metadata format. Never edit or delete a migration that has been released.

## Migrations and the baseline

New databases are built from `migrations/baseline/` instead of replaying every migration (ADR 0009); migrations added after it still run from the chain. Never edit the baseline by hand: `go run ./tools/baselinegen` regenerates it and needs a PostgreSQL and a MySQL 8 account that may create databases (`HYDRO_BASELINE_POSTGRES_URL`, `HYDRO_BASELINE_MYSQL_URL`). Only regenerate when deliberately replacing it. `go test ./internal/database/...` proves it matches the chain on SQLite; set `HYDRO_TEST_POSTGRES_URL` and `HYDRO_TEST_MYSQL_URL` to include those dialects (the PostgreSQL role must not be a superuser for the tenant RLS contract).

