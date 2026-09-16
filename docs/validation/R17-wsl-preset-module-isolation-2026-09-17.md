# R17 generated-preset module isolation — 2026-09-17

The generated presets were exercised against the migration runner for both
supported dialects.

## Command

```bash
HYDRO_TEST_POSTGRES_URL='postgres://arda@/hydro_contract?host=/var/run/postgresql&sslmode=disable' \
  go test ./tests/contract \
  -run '^TestGeneratedPresetsKeepModuleSchemasIsolated$' \
  -count=1 -p 1 -timeout=10m
```

## Result

**PASS** on WSL Ubuntu / PostgreSQL 18.6 (0.806 s) and Windows SQLite (3.210 s).
The test generates all ten shipped presets (`news`, `editorial`, `commerce`,
`compact-site`, `corporate`, `portal`, `headless`, `membership`, `dashboard`,
`erp`), applies each preset’s selected migration set in an isolated database
(or PostgreSQL schema), and checks representative module-owned tables:
Pages, Media, News, Commerce, Members and Site. Inactive module tables are
absent and enabled module tables are present; filtered compatibility also passes.

## Limitations

This closes the per-preset migration isolation evidence for both dialects. It
does not by itself certify route/asset/worker registration, customer restore,
capacity, or external deployment behavior; those remain separate release gates.
