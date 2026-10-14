# R5 SDK lockfile verification — 2026-09-16

**Status:** Successful (local/WSL)

Added bounded `hydrogo.lock` loading and verification to the project generator. `hydro project check` verifies version 1, project identity, edition/preset, Core distribution, generator version, license requirement, and duplicate/empty feature values if the lockfile is present. This operation requires no network or database connection.

## Evidence

- Windows: `go test ./internal/projectgen ./cmd/hydro` passed.
- WSL Ubuntu/Go 1.27: `go test ./...` passed.
- Generated valid lockfile was accepted.
- Tampered lockfile with empty Core version was rejected.

## Scope & Boundaries

This check covers lockfile integrity and local compatibility metadata only. Runtime entitlement, broker access, and binary signatures are verified separately.
