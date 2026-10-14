# R5 — Versioned module route contract validation

**Status:** Local contract succeeded.

`extension/v1` now exposes the stable `Module` and `ModuleRoute` contract while
retaining source compatibility with existing root-package implementations.
The package-level test verifies a public route declaration through the
versioned namespace.

Evidence: `go test ./extension/v1` passed on 2026-09-14. Full repository
pre-push verification remains the release gate.
