# R5 — Task-oriented module CLI validation

**Status:** Local contract succeeded.

`hydro module create --spec ... --out ...` now delegates to the existing
deterministic, deny-by-default collection generator. The legacy
`hydro generate module` command remains available and both paths share the
same validation and output logic.

Evidence: `go test ./cmd/hydro` passed on 2026-09-14, including the module
creation alias test. Full repository pre-push verification is required before
delivery.
