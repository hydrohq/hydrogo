# R5 runtime namespace verification — 2026-09-16

**Status:** Successful (local/WSL)

Release integrity verification is also accessible to SDK users via the `hydro runtime verify` namespace. The command reuses the existing `hydro release verify` implementation, avoiding dual verification logic.

## Evidence

- `go test ./cmd/hydro ./internal/release` passed.
- `go test ./...` passed on WSL Ubuntu/Go 1.27.
- CLI tests verify the `runtime verify` alias, accepting valid manifest/artifacts and rejecting tampered artifacts.

## Usage

```bash
hydro runtime verify --manifest release.json --public-key release.pub --root ./runtime
```

## Scope & Boundaries

This is local verification only; it does not provide downloading, login, entitlement, or artifact brokering.
