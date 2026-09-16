# R5 validation: bounded frontend manifest input

Date: 2026-09-16

- `go test ./internal/projectgen ./cmd/hydro` passes.
- Tests prove manifests over 1 MiB and route names over 128 bytes are rejected;
  symlinked path components are rejected by the path guard.
- Existing valid manifests retain deterministic route ordering and JSON output.
