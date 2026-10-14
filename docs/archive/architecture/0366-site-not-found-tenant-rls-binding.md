# ADR-0366: Bind Site not-found RLS to the canonical tenant transaction

- **Status:** Accepted
- **Date:** 2026-10-09
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0180](0180-postgres-tenant-rls-enforcement.md), [ADR-0233](0233-public-tenant-scope-contract.md), [ADR-0237](0237-tenant-rls-verification-extension.md), [ADR-0309](0309-site-not-found-tracking-and-slug-redirects.md)

## Context

The PostgreSQL policy created for `site_not_found_hits` compared `tenant_id`
with the transaction setting `hydrogo.tenant_id`. The shared tenant
transactor binds `app.tenant_rls_enforced` and `app.current_tenant_id`, as
specified by ADR-0180. Consequently, correctly scoped Site requests could not
write or read their missing-path counters under forced row-level security.
SQLite repository tests did not expose this PostgreSQL-only policy mismatch.

## Decision

- A forward-only Site migration replaces the `site_not_found_hits` RLS policy
  expression with the canonical transaction-local setting and predicate used
  by the core tenant migration.
- Shared transactions continue to see and mutate only rows matching
  `app.current_tenant_id` when `app.tenant_rls_enforced` is `true`. The
  explicitly bound single-tenant compatibility mode continues to use the
  accepted ADR-0180 behavior.
- The change preserves `ENABLE ROW LEVEL SECURITY`, `FORCE ROW LEVEL
  SECURITY`, the policy name, table schema, indexes, data, and repository
  predicates. The migration changes no SQLite or MySQL behavior.
- Policy creation remains migration-owned; runtime startup and request
  handling do not execute DDL.

## Alternatives

- Set the private `hydrogo.tenant_id` variable in Site callers: rejected
  because it would create a second tenant-binding contract and miss callers
  using the shared transaction boundary.
- Remove forced RLS or fall back to repository predicates alone: rejected
  because that would weaken the existing PostgreSQL defense-in-depth boundary.
- Edit the original migration: rejected because released migration checksums
  are immutable and deployed databases need an explicit forward correction.

## Consequences

### Benefits

- Site missing-path tracking now uses the same trusted tenant binding as every
  other PostgreSQL RLS-protected table.
- Existing rows and the table contract are unchanged.

### Costs and risks

- The forward migration must be applied before PostgreSQL Site miss tracking
  is available; migration failure leaves the previous forced policy in place.
- Direct pool access without the shared tenant transaction remains governed
  by ADR-0180's open call-site audit and is not certified by this change.

## Scale impact

- **Compact:** no new process, service, worker, route, index, or persistent
  footprint; the existing bounded per-tenant table is reused.
- **Standard:** the transaction applies one small policy replacement; no pool
  or connection behavior changes.
- **Distributed:** all instances use the same canonical transaction GUC and
  migration ledger; no process-local state is introduced.

## Database impact

- **SQLite:** no schema or query behavior changes; the migration records a
  checksum-protected no-op and existing explicit tenant predicates remain the
  boundary.
- **PostgreSQL:** an atomic, transactional policy replacement binds forced RLS
  to the standard tenant transaction settings. No data rewrite or table lock
  beyond the short policy DDL transaction is required.
- **MySQL:** no schema or query behavior changes; repository tenant predicates
  remain the boundary.
- **Future adapters:** adapters must retain the explicit tenant predicate
  contract and must not depend on PostgreSQL setting names.

The migration metadata declares the three dialect checksums, module
dependency, transaction behavior, compatibility, and forward-recovery note.

## Cache impact

No cache key, TTL, invalidation, stampede protection, fallback, or cache
service changes.

## Deployment and compatibility

The migration depends on the existing Site counter table and canonical RLS
contract, preserves stored counters, and is safe for old/new binaries because
both use the shared `app.*` transaction settings. Blue/Green binaries can
coexist while the additive migration is applied. There is no down migration;
recovery is restore or another forward policy correction. Operators should
verify migration completion before relying on PostgreSQL Site miss tracking.

## Validation

- Run `TestPostgresRedirectRuleContract` on PostgreSQL. Its not-found contract
  uses shared transactions for two tenants and proves independent writes,
  reads, counts, and deletion under forced RLS.
- Run the Site appearance registered-route CSRF/permission/persistence test
  against an isolated PostgreSQL schema to ensure the route's migration and
  database behavior compose.
- Run all `modules/site/...` tests on SQLite and PostgreSQL, plus vet and
  `migrationguard`.
- Run `go test ./...`, `go vet ./...`, moduleguard, and importguard before
  release. Production deployment and customer identity remain separate gates.

## References

- [ADR-0180: PostgreSQL tenant RLS enforcement](0180-postgres-tenant-rls-enforcement.md)
- [ADR-0233: Public tenant scope contract](0233-public-tenant-scope-contract.md)
- [ADR-0237: Edition-extensible tenant RLS verification](0237-tenant-rls-verification-extension.md)
- [Site not-found validation](../validation/site-not-found-postgres-rls-2026-10-09.md)
