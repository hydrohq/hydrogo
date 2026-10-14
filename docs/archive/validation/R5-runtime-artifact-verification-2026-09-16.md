# R5 runtime artifact verification — 2026-09-16

**Status:** Successful (local/WSL)

As a secure baseline step for private runtime distribution, `release.VerifyArtifacts` verifying all artifacts in the release manifest and the CLI option `hydro release verify --root PATH` were added. After signature verification, file size, SHA-256 digest, and escape outside root/symlinks are checked; on failure, the runtime must not be executed.

## Evidence

- `go test ./internal/release ./cmd/hydro` passed.
- `go test ./...` passed on WSL Ubuntu/Go 1.27.
- Tests verify that genuine artifacts are accepted and tampered binaries are rejected.

## Usage

```bash
hydro release verify --manifest release.json --public-key release.pub --root ./extracted-runtime
```

## Scope & Boundaries

This verifier does not perform downloads or authorization. Artifact broker, OAuth/device login, and installers are subsequent phases.
