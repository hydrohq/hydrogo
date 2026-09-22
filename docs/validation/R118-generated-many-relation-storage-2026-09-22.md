# R118 — Generated many-valued relation storage

Date: 2026-09-22

## Scope

`cardinality: many` now maps to an additive normalized relation table, not a
scalar field in the resource table. Generated repositories expose bounded,
stable-order list and atomic replace methods, with a transaction-taking variant
for Edition-owned workflows. Standard resource CRUD/query SQL excludes these
relation columns. Tenant-scoped join predicates require `tenant_id`.

## Evidence

- `go test ./internal/generator -count=1` — PASS; verifies DDL, typed model,
  scalar repository exclusion, tenant key, index and generated API.
- `go test ./examples/relation-edition/... -count=1` — PASS; executes migration
  in SQLite and exercises normal generated create, transaction participation,
  stable list order, cross-tenant isolation, duplicate rejection and item limit.
- Migration is additive and marked `expand` for SQLite/PostgreSQL. The
  generator does not apply DDL. Rollback metadata prohibits destructive down
  migration; recovery is restore/forward-only.
- The relation table receives no independent service, worker or cache.

## Boundaries and remaining certification

The Edition must authorize/tenant-check target IDs in the same owning
transaction. Foreign keys, target existence, cascade/delete behavior, eager
loading and cache invalidation are deliberately not inferred. PostgreSQL
provider execution and concurrent write/blue-green certification remain
release gates; this report does not claim them as complete.
