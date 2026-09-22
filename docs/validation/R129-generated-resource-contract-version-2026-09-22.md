# R129 — Generated resource contract version tracking — 2026-09-22

## Scope

Persist the resource-spec format version in newly generated `generator.lock`
files and surface it in read-only resource compatibility checks. Keep pre-change
lockfiles readable as v0 and reject unknown explicit versions. No schema, API,
authorization, migration execution, or route activation behavior was changed.

## Verification

- `go test ./... -p 1 -timeout=5m` — passed.
- With `HYDRO_REDIS_URL` and `HYDRO_REDIS_KEY_PREFIX` unset,
  `go test -tags hydro_no_redis ./internal/generator ./cmd/hydro -count=1` — passed.
- `hydro resource check --root` passed for Project, Article, and Relation
  Edition fixtures and reported `contract=v0` for each (37 artifacts each).
- `scripts/verify-edition-servers.ps1` passed both SQLite Edition migration,
  health, POST, and GET smoke flows. `scripts/verify-edition-footprint.ps1`
  passed its full matrix; the Compact combined Workspace measured 21,343,744
  bytes and 152 non-standard packages on Windows amd64/Go 1.27.0.
- Tests cover emitted lock metadata, legacy lock acceptance without the marker,
  and rejection of an unsupported contract version.
- `git diff --check` — passed.

## Compatibility and limits

The persisted resource contract remains v0. Adding v1 semantics still requires
the versioned loader/schema conversion and compatibility evidence described by
[ADR-0241](../architecture/0241-versioned-blueprint-input.md). No commercial
Edition, live migration, PostgreSQL, or cross-version deployment certification
is claimed by this local test result.
