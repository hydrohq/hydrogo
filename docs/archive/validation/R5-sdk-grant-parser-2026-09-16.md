# R5 SDK grant parser — 2026-09-16

**Status:** Successful (local/WSL)

`internal/sdkdist.ParseGrant` limits the broker response body to 64 KiB, rejects trailing data in JSON, and prepares the grant for entitlement validation. Tokens are not retained or logged; `DownloadGrantAndInstall` must be called after `ValidateGrant`.

## Evidence

- Windows: `go test ./internal/sdkdist` passed.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` passed.
- A full `go test ./...` was run on WSL following changes; CLI observation timed out while packages continued testing, so this report closes only the completed `internal/sdkdist` evidence. Previous full-suite evidence is preserved in `R5-sdk-grant-download-integration-2026-09-16.md`.
- Oversized and trailing JSON grant responses were rejected.

## Scope & Boundaries

This parser does not handle OAuth/device flow, token issuance, or revocation.
