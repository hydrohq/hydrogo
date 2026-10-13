# R5 SDK project check command — 2026-09-16

**Status:** Successful (local/WSL)

`hydro project check` was added to the public SDK workflow as a dependency-free project verification step. It reuses existing `hydro validate` logic, checking configuration, profile, and module invariants without opening database connections.

## WSL evidence

```bash
cd /path/to/project
go test ./...
go run ./cmd/hydro project check --config /tmp/hydro-project-check/config.yaml --json
```

Actual result:

```json
{"valid":true,"profile":"compact","database":"sqlite","modules":["core","pages"],"cache_entries":256,"queue_entries":10000}
```

## Scope & Boundaries

This command does not apply migrations, start runtimes, or grant private artifact access. Its purpose is to surface misconfigurations early and safely prior to running `hydro dev` or `hydro migrate`.
