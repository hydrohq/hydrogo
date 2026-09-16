# R5 validation: standalone frontend manifest check

Date: 2026-09-16

- `go test ./cmd/hydro ./internal/projectgen` passes.
- The CLI test validates a manifest, reports the route count, and confirms
  deterministic alphabetical ordering in `--json` output.
- Existing `hydro new --frontend-manifest` behavior continues to use the same
  validator.
