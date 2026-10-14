# Development

## Run it

```bash
go run ./tools/devserver
```

This rebuilds on save, seeds disposable demo data and serves the site at
http://127.0.0.1:8090 (admin at `/admin`). Use `-address 0.0.0.0:8090` to
open it from a phone on the same network. The public theme defaults to
`corporate`; run `HYDRO_THEME=newsroom go run ./tools/devserver` for the news
theme. (`hydro serve --config config.yaml` reads `theme:` from the file.)

Optional local PostgreSQL/Redis: `docker compose -f compose.dev.yaml up -d`.

## Check it

```bash
go run ./tools/check          # or: make check
go run ./tools/check -short   # skip tests
```

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

- Templ: edit `.templ`, then `go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate`.
- Theme and admin CSS: edit the `*.source.css` / `input.css` file, then run the
  matching `scripts/build-*-tailwind.ps1` with the standalone Tailwind CLI.

## Migrations

Add a directory `migrations/<YYYYMMDDNNNN>_<name>/` with `migration.yaml`,
`sqlite.sql`, `postgres.sql` and `mysql.sql`. Look at a recent one for the
metadata format. Never edit or delete a migration that has been released.
