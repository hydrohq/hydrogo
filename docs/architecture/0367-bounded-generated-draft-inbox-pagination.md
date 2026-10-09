# ADR-0367: Bounded generated draft inbox pagination

- **Status:** Accepted
- **Date:** 2026-10-10
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0209](0209-typed-server-ui-contract.md), [ADR-0229](0229-generated-list-transport-contract.md), [ADR-0235](0235-generated-tenant-scoped-resource-contract.md), [ADR-0272](0272-admin-studio-business-ux.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md)

## Context

The generated editorial draft inbox called `ListDrafts` with a fixed limit of
50, failed when a callback returned more than 50 rows, and exposed shared-list
sort links without asking Edition storage to sort. Editors could not navigate
large draft collections reliably. Existing generated collection lists already
use bounded, stable pagination; the editorial inbox should provide the same
basic navigation without making Core query Edition-owned storage.

## Decision

- Add an optional Edition callback `ListDraftPage` that receives the request,
  collection, locale, offset, and limit. Edition callbacks remain responsible
  for authorization, tenant scope, and storage access.
- The inbox requests 25 rows plus one lookahead row. It validates the page
  query (1–100), bounds offsets to the resulting range, trims the lookahead,
  and renders localized Previous/Next controls on the existing `/drafts` route.
- Edition callbacks order drafts by `updated_at DESC, resource_id DESC` (or an
  equivalent stable order) before applying offset/limit. Sort/search/filter
  controls are not shown until Edition-owned draft query contracts support
  those operations.
- Keep the existing `ListDrafts` callback as a compatibility path. When only
  that callback is supplied, the inbox keeps its current first-page behavior
  and renders no pagination controls. A request for page > 1 without
  `ListDraftPage` is rejected; it never repeats page one as page two.
- A draft-enabled resource may provide either callback. The new paged callback
  may replace `ListDrafts` when an Edition has migrated its storage adapter.

## Alternatives

- Keep failing above 50 rows: rejected because editors cannot reach or triage
  the remaining drafts.
- Fetch all drafts and paginate in Core: rejected because result size and
  memory use would be unbounded, and Core would still need a full Edition query.
- Change the existing callback signature: rejected because it breaks Edition
  source compatibility unnecessarily.
- Show sort/search/filter links before the storage callback supports them:
  rejected because controls must reflect actual behavior.

## Consequences

### Benefits

- Draft lists remain bounded and editors can navigate up to 100 pages.
- Older Edition compositions continue to compile and retain their first-page
  behavior until they opt into the paged callback.
- Pagination state and localized content locale remain on the existing Admin
  route; no client runtime or new route is introduced.

### Costs and risks

- Editions must provide a stable, tenant-scoped paged query to navigate beyond
  the legacy first page.
- Offset pagination can become slower at high offsets; the hard 100-page cap
  bounds work. A keyset contract requires measured need and a separate decision.
- Database adapter tests must verify ordering and page boundaries on each
  supported dialect used by an Edition.

## Scale impact

- **Compact:** no new service, worker, cache, or browser dependency. Each
  request returns at most 26 draft documents.
- **Standard:** uses the Edition's existing database pool and bounded query.
- **Distributed:** handler remains stateless. Every page request resolves
  identity, policy, locale, and tenant scope through Edition callbacks.

## Database impact

- **SQLite:** Edition callback uses bound `LIMIT` and `OFFSET` with stable
  ordering; no Core migration is added.
- **PostgreSQL:** Edition callback uses the corresponding bound parameters and
  the same order; no Core migration is added.
- **Future adapters:** preserve the Edition callback contract and certify the
  same stable page ordering/boundary behavior.

## Cache impact

Draft inbox responses remain private and non-cacheable. No cache key, TTL,
invalidation, or cache service is introduced.

## Deployment and compatibility

The callback addition is optional and source-compatible. Legacy callbacks
continue to render their existing first page; new Edition code may supply only
`ListDraftPage`. An Edition switching from legacy to paged storage must deploy
the updated composition with the matching Core version. There is no migration;
rollback restores the prior binary and prior callback composition.

## Validation

- Handler tests cover page-one lookahead, page-two offset, localized
  Previous/Next controls, page bounds, callback output bounds, and legacy
  first-page compatibility.
- Edition storage contract tests cover stable order and SQLite/PostgreSQL
  pagination boundaries under tenant scope.
- Browser acceptance verifies page navigation, keyboard operation, locale
  preservation, and axe. Responsive viewport coverage remains part of the
  shared browser acceptance gate.

## References

- [ADR template](ADR_TEMPLATE.md)
- [ADR-0229: Generated bounded list transport](0229-generated-list-transport-contract.md)
- [ADR-0346: Blueprint Admin Studio and editorial workflows](0346-blueprint-admin-studio-workflows.md)
