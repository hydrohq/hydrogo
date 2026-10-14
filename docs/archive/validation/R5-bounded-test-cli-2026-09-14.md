# R5 — Bounded test CLI validation

**Status:** Local contract succeeded.

`hydro test` now provides package selection, bounded timeout, repeat count, and
optional race detection while passing arguments directly to `go test` without a
shell. Invalid bounds are rejected before a child process is launched.

Evidence: `go test ./cmd/hydro` passed on 2026-09-14, including invalid-bound
coverage. Full repository pre-push verification is required before delivery.
