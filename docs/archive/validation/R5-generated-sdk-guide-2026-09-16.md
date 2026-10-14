# R5 generated SDK guide verification — 2026-09-16

**Status:** Successful (local/WSL)

Added `docs/SDK.md` to `hydro project new` output. The document explains lockfiles, local workflows, runtime distribution boundaries, and extension boundaries for `custom/`, `themes/`, and `migrations/` upon project generation.

## WSL evidence

```bash
cd /path/to/project
go test ./internal/projectgen ./cmd/hydro ./internal/generator ./internal/release ./internal/sdkdist
go run ./cmd/hydro project new sdk-doc --preset compact-site --no-demo --root /tmp/hydro-sdk-doc-check
test -f /tmp/hydro-sdk-doc-check/sdk-doc/docs/SDK.md
```

Actual result: generator and CLI tests passed; `sdk-doc/docs/SDK.md` generated (`sdk-doc-ok`).

## Scope & Boundaries

This document does not implement runtime artifact brokers or installers; it explicitly notes within generated projects that these components are not yet implemented.

The current template also illustrates `hydro check`, `resource new/check`, `permissions check`, `api diff`, and signed `schema diff` steps. Following template changes, `go test ./internal/projectgen` passed on Windows; full WSL suite is retained as separate cross-platform evidence.
