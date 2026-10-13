# R5 SDK runtime cache contract — 2026-09-16

**Status:** Successful (local/WSL)

`internal/sdkdist.CachePath` deterministically generates runtime artifact cache paths based on version/product/OS/architecture. Version and product inputs are sanitized against path traversal; the function neither creates directories nor executes binaries.

## Evidence

- Windows: `go test ./internal/sdkdist` passed.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` and `go test ./...` passed.
- `../`, `..`, empty, and slash-containing version values were rejected.

## Scope & Boundaries

This is a cache path contract only. Downloading, atomic installation, cache eviction, version switching, and broker integration are subsequent phases.
