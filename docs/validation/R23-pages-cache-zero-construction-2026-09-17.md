# R23 — Pages cache zero-construction guard (2026-09-17)

The server bootstrap now allocates the in-process Pages cache and optional
Redis two-tier page cache only when the `pages` module is enabled. Metrics
export remains safe for inactive profiles and reports zero cache counters
instead of dereferencing a missing cache.

This prevents inactive/headless configurations that do not expose Pages from
consuming page-cache memory or attempting page-cache Redis wiring. The existing
module route and migration filters remain in force.

Verification: `go test ./cmd/hydro ./internal/projectgen -count=1` and
`git diff --check` — PASS.
