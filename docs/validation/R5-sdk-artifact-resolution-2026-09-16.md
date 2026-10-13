# R5 SDK artifact resolution — 2026-09-16

**Status:** Successful (local/WSL)

`internal/sdkdist.ResolveRuntimeArtifact` uniquely selects the canonical runtime artifact for the target OS/architecture from the signed release manifest. Missing, duplicate, unsafe path, or empty checksum metadata is rejected fail-closed. Signature and file integrity verification are handled by `release.Verify` and `release.VerifyArtifacts` respectively.

## Evidence

- Windows: `go test ./internal/sdkdist` passed.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` and `go test ./...` passed.
- Linux/Windows artifact selection and rejection of invalid metadata / missing artifacts were tested.

## Scope & Boundaries

This layer performs no network access, login, entitlement, or file downloading; it is the deterministic target resolution contract for the broker and installer.
