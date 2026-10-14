# R5 SDK bounded artifact download — 2026-09-16

**Status:** Successful (local/WSL)

`internal/sdkdist.DownloadAndInstall` downloads the runtime artifact from the broker-supplied URL to a temporary file using a bounded response body. Redirects are not followed, non-200 HTTP responses are rejected, size limits are enforced, and checksum-verified atomic installation runs afterward.

## Evidence

- Windows: `go test ./internal/sdkdist` passed.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` and `go test ./...` passed.
- The correct artifact was downloaded and installed from a local HTTP test server.
- Redirects and exceeded configured size limits were rejected.

## Scope & Boundaries

This client does not perform login or token issuance; ensuring the URL is authorized by the broker is the caller's responsibility. In production, only HTTPS URLs and short-lived broker authorization should be used.
