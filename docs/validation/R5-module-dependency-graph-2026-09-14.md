# R5 — Module dependency graph validation

**Status:** Local contract succeeded.

The versioned SDK now rejects unknown dependencies and cycles and normalizes
module identifiers deterministically. Unit coverage includes an acyclic graph,
unknown dependency, and cycle.

Evidence: `go test ./extension/v1` passed on 2026-09-14. Full pre-push
verification remains the release gate.
