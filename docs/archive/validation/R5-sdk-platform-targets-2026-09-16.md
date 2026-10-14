# R5 SDK platform targets — 2026-09-16

**Status:** Successful (local/WSL)

`internal/sdkdist` added the shared platform contract used by brokers and installers. Linux/Windows/macOS and amd64/arm64 targets are normalized; `.exe` extension is appended on Windows; unknown platforms, architectures, or product names containing paths are rejected fail-closed.

## Evidence

- Windows: `go test ./internal/sdkdist` passed.
- WSL Ubuntu/Go 1.27: `go test ./internal/sdkdist` and `go test ./...` passed.
- Tests verify Linux/Windows/macOS naming conventions and rejection of unsafe/unsupported targets.

## Example names

```text
hydro-linux-amd64
hydro-linux-arm64
hydro-windows-amd64.exe
hydro-darwin-arm64
```

## Scope & Boundaries

This package does not download artifacts, sign files, or publish installers; it provides only the canonical target/naming contract across all distribution components.
