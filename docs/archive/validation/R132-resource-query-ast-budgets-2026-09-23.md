# R132 — Resource query AST validation and budgets — 2026-09-23

## Scope

Aligned the internal Core parser and public Edition `extension/v1` parser and
SQL builders. Added caps for query key count/input bytes, `in` members, total
bound arguments, sort fields, and pagination. Rejected repeated scalar values,
invalid `exists`, simultaneous `page`/`offset`, and empty `in` members. SQL
builders now revalidate direct ASTs, including identifiers and bounds, rather
than assuming every caller used a parser.

## Verification

- `go test ./internal/query ./extension/v1` — passed.
- `go test ./... -p 1 -timeout=5m` — passed, including the full repository
  suite and SQLite-backed contract package.
- `go vet ./...` — passed.
- `git diff --check` — passed.

## Limits

These checks bound query construction but do not certify query-specific index
plans, tenant/RLS policies, database statement timeouts, PostgreSQL live
behavior for Edition-generated repositories, or performance under production
data. No dedicated query benchmark is claimed by this report.
