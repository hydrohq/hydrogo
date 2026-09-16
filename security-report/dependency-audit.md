# Dependency and supply-chain audit — 2026-09-16

## Evidence

Commands run from the repository root:

```text
go mod verify
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
go vet ./...
```

All commands passed. `govulncheck` reported **0 vulnerabilities reachable by
the code**. Its informational output mentions vulnerabilities in one imported
package and one required module that are not called by this code; these remain
vendor advisories to monitor rather than confirmed application findings.

The same `go mod verify`, `govulncheck`, `go vet`, and full test suite also
passed independently on Ubuntu/WSL2 with Go 1.27.0 linux/amd64.

The repository also generates a deterministic SPDX SBOM and pins GitHub
Actions. Runtime image build and registry scanning require an OCI-enabled CI or
deployment environment and are recorded as `needs_validation`.
