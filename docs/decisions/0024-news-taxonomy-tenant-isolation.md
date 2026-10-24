# 0024: News taxonomy had no tenant isolation at all

- Status: Accepted
- Date: 2026-10-24

## Context

A final, high-stakes re-assessment (security/performance/maintenance/
reliability, four parallel reviews) run before committing to this repo as
a long-term foundation found one high-severity, genuinely exploitable gap:
`news_taxonomy_terms` and `news_taxonomy_relations` (categories and tags
for news articles, `202609160001_news_taxonomy`) never got a `tenant_id`
column at all, unlike `news_items` itself (`202609190007_news_tenant_scope`)
or every other tenant-scoped table (`202609200003_postgres_tenant_rls`'s
RLS table list). `modules/news/repository/sqlstore/taxonomy.go` had zero
references to `internal/tenant` -- every query read and wrote across all
tenants in a shared database.

This was worse than a read-only leak. `POST /api/v1/admin/news/taxonomy`
accepts a client-supplied `id` and the existing code did a bare
`INSERT ... ON CONFLICT(id) DO UPDATE`, keyed only on that global id. Any
authenticated admin in Tenant B who sent Tenant A's term id (observable
from Tenant A's own public content API, or simply guessable for a
sequential/slug-derived id scheme) would silently overwrite Tenant A's
term. `DELETE /api/v1/admin/news/taxonomy/{id}` had the same shape: no
tenant check at all, so Tenant B could delete Tenant A's term outright,
cascading to strip Tenant A's articles of their categories and tags via
the relations table's `ON DELETE CASCADE`.

The six-phase audit's phase 2 route-permission work did not catch this
because it verified *that* a declared `Permission` is enforced -- it
never audited *which* routes declare none, and this is not really a
role/permission bug at all: a correctly-scoped admin role in the right
tenant would hit the exact same cross-tenant corruption. The bug is a
missing tenant discriminator in the data model, found only by reading
`taxonomy.go` end-to-end against the tenant-scoping convention every
sibling file in the same package already follows.

## Decision

Two additive migrations, following the exact pattern already established
for `news_items` and the rest of the RLS table list:

- `202610241800_news_taxonomy_tenant_scope`: `tenant_id` column
  (default-empty, matching every other tenant-scope migration) plus a
  supporting index on each table.
- `202610241801_news_taxonomy_rls`: PostgreSQL RLS for both tables --
  `202609200003_postgres_tenant_rls` is never edited once released
  (AGENTS.md rule 3), so the two tables it missed get their own migration
  instead, same policy shape.

`modules/news/repository/sqlstore/taxonomy.go` was rewritten so every
method resolves the caller's tenant and scopes every statement by it.
`UpsertTaxonomyTerm` in particular no longer does a bare
`ON CONFLICT(id) DO UPDATE`: it tries an `UPDATE ... WHERE id=? AND
tenant_id=?` first, and only inserts a new row if no row with that id
exists for *any* tenant -- a client-supplied id that already belongs to
another tenant now fails closed with a new `ErrTaxonomyTermIDTaken`
(mapped to `409 Conflict`) instead of silently overwriting. `DeleteTaxonomyTerm`
and the child/relation-count checks it depends on are scoped the same way.

`tests/contract/postgres_rls_contract_test.go`'s `expectedRLSTables` list
(the structural contract proving every table in it has RLS enabled,
forced, and the isolation policy present) gained both tables. A new
`taxonomy_tenant_contract_test.go` proves the actual cross-tenant
behavior end to end on both SQLite and PostgreSQL (including against a
non-superuser role, since a superuser bypasses RLS and would make the
Postgres half of this a no-op test): a term created under one tenant is
invisible to another tenant's catalog, cannot be overwritten by reusing
its id, cannot be deleted, and its relations do not leak through
`ListTaxonomy`.

Separately, the same review round found `cmd/hydro/serve_runtime.go`'s
IP-blocklist refresh goroutine bypassing `serveApp.spawn`/`waitForWorkers`
(the only background worker that did), and the admin Members list query
having no supporting index for its actual `(tenant_id, status, created_at
DESC)` shape, unlike every sibling list (News, Pages). Both are now fixed
(`a.spawn(...)`; `202610241802_members_list_index`) -- low severity, worth
fixing in the same pass rather than opening three separate narrow commits.

## Consequences

- `go run ./tools/check` passes. The new contract test runs on SQLite, on
  a Postgres superuser connection (proves the app-layer tenant filtering,
  independent of RLS), and on the non-superuser `hydro_rls_test` role
  (proves RLS itself, since a superuser bypasses it regardless of
  `FORCE ROW LEVEL SECURITY`). The existing `TestPostgresTenantRLSContract`
  structural test also passes with both tables added to its list.
- Verified against real MySQL 8 and with `-race`, both via this machine's
  WSL2 Docker setup, for every package touched.
- This is the second time this kind of gap (a module adding its own table
  without threading it through every tenant-scope mechanism the rest of
  the codebase already has) has surfaced in review rather than being
  caught when the table was first added. Worth a standing habit, not
  automated here: when a new table is added to any tenant-scoped module,
  check it against `tests/contract/postgres_rls_contract_test.go`'s list
  and the sibling repository file's `tenantFilter`/`RequireTenantScope`
  pattern in the same review, not as a follow-up.
