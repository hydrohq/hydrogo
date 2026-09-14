# R5 — Offline configuration validation validation

**Status:** Local contract succeeded.

`hydro validate` now validates configuration, profile budgets, and normalized
module names without connecting to the database or requiring runtime services.
It supports deterministic text and JSON output and rejects unexpected
arguments through the standard CLI parser.

Evidence: `go test ./cmd/hydro` passed on 2026-09-14, including a test using a
missing SQLite path to prove no database connection is required. Full
repository pre-push verification is required before delivery.
