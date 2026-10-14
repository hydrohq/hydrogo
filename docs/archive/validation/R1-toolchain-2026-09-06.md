# R1 toolchain initialization record

**Date:** 2026-09-06  
**Roadmap item:** R1 — Go module/toolchain and working skeleton initialization  
**Status:** Partial evidence; S01–S04 not completed.

## Environment

- Development OS: Windows.
- Go: Go Programming Language 1.27.0, installed via Winget package `GoLang.Go`.
- Module path: `github.com/ardaozyildirim/hydrogo`.
- Initial dependencies: Go standard library only.

## Expected behavior

`go test ./...`, `go vet ./...` and `go build ./cmd/hydro` must succeed. The development CLI must offer `version` and minimal `serve` commands; `/healthz` and `/readyz` must be explicitly registered, an unknown route must return 404.

## Evidence

On 2026-09-06, the following checks succeeded on Windows:

```text
go version
go test ./...
go vet ./...
go build -o bin/hydro.exe ./cmd/hydro
hydro version
```

`go test ./...` validated two packages: CLI command parsing and only explicitly registered operational routes. The CI workflow repeats the same format, vet, test, and build checks on Linux.

## Boundaries

This skeleton does not include DB connection or migration, config file, session/bootstrap, CSRF/audit, Client Edition import boundary, or Templ/CSP evidence. `/readyz` currently only represents in-process startup readiness; DB readiness will not be declared until R1's DB experiment is complete.
