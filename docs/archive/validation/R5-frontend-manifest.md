# R5 validation: versioned frontend manifest

Date: 2026-09-14

- `go test ./internal/ui` passes.
- Manifest validation rejects missing versions, relative paths, unsupported
  methods, ambiguous access classes, duplicate names, and undocumented
  selectors.
- The contract is descriptive; router authorization and CSRF remain separate
  runtime boundaries.
