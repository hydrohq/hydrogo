# R5 — Semantic UI token contract validation

**Status:** Local contract documented.

The shared token vocabulary and accessibility invariants are documented for
Admin, member, and public themes. Existing Admin Studio CSS already exposes the
reference token groups and reduced-motion override.

Evidence: `go test ./internal/ui` passed on 2026-09-14. Visual regression and
cross-browser checks remain open UI tasks.
