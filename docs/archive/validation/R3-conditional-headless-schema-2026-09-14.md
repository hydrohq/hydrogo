# R3 — Conditional headless schema validation

**Status:** Local contract succeeded; real browser and PostgreSQL deployment
certification remain open.

The GraphQL root schema now includes only fields backed by enabled modules.
Pages-only exposes `page` but not `products`; Commerce-only exposes
`products` but not `page`. Bootstrap passes the active Commerce capability so
the catalog resolver is no longer silently disabled. The endpoint remains
read-only and retains its existing request-size, complexity, and rate limits.

Evidence: `go test ./modules/api/graphql ./cmd/hydro` and the repository
pre-push verification (`scripts/verify-local.ps1`) passed on 2026-09-14.
