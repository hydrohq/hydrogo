# R5 — Canonical module filtering validation

**Status:** Local contract succeeded.

Migration selection now accepts mixed-case and whitespace-padded enabled module
names consistently with the runtime registry. A SQLite MapFS fixture verifies
that the module migration is selected.

Evidence: `go test ./internal/database` passed on 2026-09-14.
