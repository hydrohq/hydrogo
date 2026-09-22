# ADR-0240: Generated resources expose explicit route composition

- **Durum:** Accepted
- **Tarih:** 2026-09-22
- **Kapsam:** Generated collection/global/tenant-scoped resource runtime wiring
- **İlgili:** ADR-0010, ADR-0038, ADR-0229, ADR-0233

## Karar

Every generated resource emits a `Register<Name>Route` helper that composes its
generated repository, service, and HTTP transport on a standard-library
`http.ServeMux`. The helper is dependency-injected and fail-closed: it requires
the database, policy, and (for tenant-scoped resources) the Edition-owned tenant
resolver.

The helper does not execute migrations, invent authorization, resolve tenants,
start workers, install middleware, or add a route to a running server by
itself. The Client Edition/module composition root calls it explicitly and
continues to own lifecycle, CSRF, request limits, audit, cache policy, and
transaction semantics. Inactive resources therefore retain zero runtime
footprint until their Edition enables them.

## Consequences

- A Blueprint now produces a directly reusable route-composition primitive,
  reducing manual wiring without moving business policy into Core.
- Generated `RUNTIME_WIRING.md` points to the helper and retains the review
  checklist for migration, policy, capacity, and deployment evidence.
- Existing Editions can migrate incrementally; the existing constructors and
  handlers remain source-compatible.
- `ServeMux` registration is an adapter convenience, not a replacement for
  the repository's router/security middleware contract.

## Verification

The generator contract tests assert deterministic `runtime.go` output,
required dependency/policy guards, tenant resolver support, and the route
wiring checklist. Generated code remains migration-free and policy-neutral.
