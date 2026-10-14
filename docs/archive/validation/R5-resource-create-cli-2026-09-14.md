# R5 — Resource create CLI validation

**Status:** Local contract succeeded.

`hydro resource create` accepts the same versioned collection spec as
`hydro module create` and emits the same generated CRUD artifacts without
implicitly enabling a runtime module.

Evidence: `go test ./cmd/hydro` passed on 2026-09-14. Full pre-push verification
remains the release gate.
