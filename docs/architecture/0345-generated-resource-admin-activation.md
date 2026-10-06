# ADR-0345: Generated resource Admin Studio activation

- **Status:** Accepted
- **Date:** 2026-10-07
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0211](0211-declarative-module-resource-contract.md), [ADR-0223](0223-generated-admin-ui-and-publishing-preview.md), [ADR-0234](0234-generated-admin-view-metadata.md), [ADR-0240](0240-generated-route-composition.md)

## Context

Blueprint generation currently emits typed Admin list/form templates and a
presentation manifest. It also emits an API route registration helper. These
artifacts are not connected to the Client Edition's Admin Studio route table or
navigation, so a generated resource does not appear as a usable panel without
manual Edition wiring.

The activation step must remove repetitive UI and route wiring while preserving
the Edition's ownership of authorization, tenant resolution, validation,
middleware, audit, migration execution, and resource lifecycle. A Blueprint
must not gain access merely by declaring presentation metadata.

## Decision

The generator will provide a deterministic Edition-level registry artifact
from generated resource specs. The Edition composition root consumes that
registry through one explicit activation call. The shared Admin Studio renders
list/create/edit screens from this metadata and forwards reads and mutations
to the Edition's already-composed resource API handlers.

The Edition supplies the authenticated-admin check, CSRF validation and page
shell. Resource authorization, tenant scope, validation and audit remain in
the resource API handler. An Edition must compose the generated API route for
each resource it exposes in the registry. The generator must not invent
permissions, add role grants, execute migrations or register API routes.

The activation artifacts are build-time source/metadata, not runtime directory
scanning. Existing generated API registration remains source-compatible.

## Alternatives

- Runtime scan of generated directories: rejected because it makes deployment
  contents an implicit route-registration authority and obscures inactive
  resource behavior.
- Hand-written route and sidebar registration for every resource: rejected
  because it preserves the current repetitive failure point.
- Blueprint-granted default administrator permissions: rejected because
  presentation and schema metadata cannot define Edition authorization policy.

## Consequences

### Benefits

- A generated resource can become a complete Admin CRUD surface through one
  reviewed Edition activation point.
- Navigation and screens derive from the same versioned generated metadata.
- Security and lifecycle ownership remain explicit and testable.

### Costs and risks

- The registry and activation contract become versioned generated artifacts.
- Editions must provide the resource policy and tenant resolver before enabling
  a resource.
- Generated Admin handlers add bounded request parsing and require contract
  tests for CSRF, authorization, validation, and tenant isolation.

## Scale impact

- Compact: the shared handler introduces no worker or cache. Form and API
  response bodies are bounded, and list requests are paginated.
- Standard: no worker or cache is introduced; normal Edition pool limits apply.
- Distributed: authorization and tenant scope are resolved per request by the
  Edition; generated handlers are stateless.

## Database impact

- SQLite: use the generated repository contract and certified SQLite tests.
- PostgreSQL: use the same logical repository contract and certified
  PostgreSQL tests.
- Future adapters: no dialect-specific behavior may escape the generated
  repository boundary.

## Cache impact

Generated Admin pages are private and non-cacheable by default. No new cache,
key, TTL, invalidation, or stampede behavior is introduced.

## Deployment and compatibility

Registry output is deterministic and build-time. No runtime DDL or directory
scan is permitted. Old binaries retain their existing route set; a new binary
activates only resources included in its compiled Edition registry. Rollback
uses the previous binary and existing migration compatibility procedures.

## Validation

- Generator tests verify deterministic aggregate registry artifacts,
  including resource metadata and global-resource views.
- Generated Admin handler tests verify authorization, CSRF, bounded requests,
  list/form rendering and delegation to resource APIs. The Edition composition
  test exercises generated CRUD over SQLite.
- `moduleguard`, `migrationguard`, and `importguard` remain mandatory.

## References

- `docs/architecture/ADR_TEMPLATE.md`
- `AGENTS.md` contribution and definition-of-done requirements
