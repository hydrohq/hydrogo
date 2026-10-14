# ADR-0249: Optional GraphQL runtime build capability

- **Status:** Accepted
- **Date:** 2026-09-22
- **Related:** ADR-0010, ADR-0143, ADR-0144, ADR-0247, ADR-0248

## Context

The universal Workspace runtime imported GraphQL parsing and schema packages
even when an Edition only used the versioned REST API. Runtime module
activation already controlled whether `/api/graphql` was registered, but did
not control whether its implementation and dependency tree were linked into
the customer artifact. This raised binary and supply-chain surface for
Editions that did not use GraphQL.

## Decision

The default build retains the GraphQL handler. An Edition can explicitly omit
it with the `hydro_no_graphql` build tag; in that build the GraphQL route is
not registered and the GraphQL handler dependency is absent from the
transitive import graph. Customer package builders accept
`HYDRO_GRAPHQL=enabled|disabled`, defaulting to `enabled`, and record the
choice in `PACKAGE-MANIFEST.md`.

The feature selector does not change the REST API, schema, migrations, or
module contracts. It is a composition-time capability, not runtime plugin
loading. GraphQL-dependent Editions must be built with it enabled. This ADR
does not imply that GraphQL can be safely omitted from all future Editions;
the Edition's declared API contract determines the correct selection.

## Alternatives

- Always link GraphQL and only conditionally register the route: preserves
  binary flexibility but does not reduce the dependency/supply-chain surface.
- Runtime plugin loading: rejected because it compromises the self-contained,
  statically composed Edition artifact and complicates provenance.
- Remove GraphQL globally: rejected because it breaks current supported API
  behavior and is not justified by a footprint-only objective.

## Consequences

### Benefits

- GraphQL-free artifacts omit the GraphQL implementation and parser packages.
- Default builds preserve existing behavior.
- CI proves the route factory is disabled and rejects GraphQL dependencies in
  the selected Compact variant.

### Costs and risks

- Capability selection is fixed at build time and must match Edition API
  requirements.
- The Workspace Compact binary still has many optional dependencies; this
  single feature removal is not proof of complete Core composition or a
  production-size target.

## Scale impact

- Compact: no extra services or goroutines; opt-out removes a bounded query API.
- Standard: same artifact choice; GraphQL stays linked by default.
- Distributed: Edition selects GraphQL according to its API contract.

## Database impact

No changes to SQLite, PostgreSQL, migrations, transactions, or future adapter
contracts.

## Cache impact

None; cache keys, TTLs, invalidation, stampede protection, and fallback are
unchanged.

## Deployment and compatibility

No schema migration is introduced. Default dual-driver builds keep GraphQL.
An artifact without GraphQL does not register `/api/graphql`; deploying it is
an API capability change and must be coordinated with consumers. Rollback is
to the previous artifact. Blue/green compatibility requires both versions to
serve the Edition's declared API contract during the transition.

## Validation

CI builds default and `hydro_no_graphql` runtime variants, tests the handler
capability for both selections, and verifies the Compact SQLite/no-GraphQL
artifact has no GraphQL packages. Current measurements are recorded in R124.

## References

- [SQLite-only customer runtime build variant](0248-sqlite-only-customer-runtime-build.md)
- [Core and optional runtime dependency measurements](../validation/R124-no-graphql-runtime-variant-2026-09-22.md)
