# R5 — Migration create scaffold validation

**Status:** Local contract succeeded.

The CLI creates non-overwriting SQLite/PostgreSQL placeholders and a complete
metadata skeleton. A follow-up migrationguard run intentionally remains
required after SQL and checksums are reviewed.

Evidence: `go test ./cmd/hydro` passed on 2026-09-14. Full pre-push verification
remains the release gate.
