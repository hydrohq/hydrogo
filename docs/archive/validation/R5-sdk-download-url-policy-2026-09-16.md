# R5 SDK artifact download URL policy — 2026-09-16

**Status:** Successful (local/WSL)

`internal/sdkdist.ValidateDownloadURL` checks broker-supplied artifact URLs enforcing HTTPS and host presence, rejecting credentials and URL fragments, and validating scheme allowlists. HTTP is only permitted in explicitly enabled localhost development scenarios.

## Evidence

- Windows: `go test ./internal/sdkdist` passed.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` and `go test ./...` passed.
- Remote HTTP, `file:` URLs, credentials, fragments, and malformed URLs were rejected.
- Explicit localhost HTTP development exception was accepted.

## Scope & Boundaries

This policy does not make HTTP requests, generate tokens, or follow redirects. Download clients and broker authentication are subsequent phases.
