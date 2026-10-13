# R5 SDK grant → download → install integration — 2026-09-16

**Status:** Successful (local/WSL)

`DownloadGrantAndInstall` does not initiate HTTP requests without validating the broker grant against project/Core/SDK versions and expiry. Following successful validation, the bounded download, checksum verification, and atomic cache install pipeline executes.

## Evidence

- Windows: `go test ./internal/sdkdist` passed.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` and `go test ./...` passed.
- Installation succeeded with a valid grant from a local broker-like HTTP server.
- Project mismatch grant was rejected prior to initiating network requests.

## Scope & Boundaries

OAuth/device login, grant issuance, token revocation, and entitlement services are not implemented. This function serves as the secure client entry point for broker-issued grants.
