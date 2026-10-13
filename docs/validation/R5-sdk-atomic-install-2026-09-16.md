# R5 SDK atomic runtime install — 2026-09-16

**Status:** Successful (local/WSL)

`internal/sdkdist.InstallArtifact` first copies the verified runtime artifact to a temporary file in the same cache directory, followed by fsync and a secondary checksum verification before atomic rename. An existing identical artifact is reused; overwriting with a different file is prohibited. Source/destination symlink and regular-file checks are enforced.

## Evidence

- Windows: `go test ./internal/release ./internal/sdkdist` passed.
- WSL Ubuntu/Go 1.27: `go test ./internal/release ./internal/sdkdist` and `go test ./...` passed.
- Tests verify valid installation, idempotent re-run, tampered source rejection, and rejection of overwriting a different destination.

## Scope & Boundaries

This operation does not perform network downloading or broker authorization. Ensuring the source file was downloaded with manifest metadata is the caller's responsibility; checksum and atomic installation are re-guaranteed here.
