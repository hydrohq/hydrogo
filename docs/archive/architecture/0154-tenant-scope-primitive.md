# ADR-0154: Explicit tenant scope primitive

- Status: Accepted (foundation only; shared multi-tenant mode remains disabled)
- Date: 2026-09-19
- Related decisions: [ADR-0140](0140-tenant-isolation-scope.md), [ADR-0149](0149-shared-multi-tenant-saas-evolution.md)

## Context

ADR-0140 requires tenant scope to be explicit and fail closed, but the current
multi-tenant contract test carried its own mock context type. Shared adapters
need one production-safe primitive before scope can be propagated through
routers, services, repositories, jobs, media, cache, audit, and restore flows.

## Decision

Add `internal/tenant` with a validated `Scope`, explicit `WithScope`/
`Require` context operations, and deterministic `CacheKey` generation. Tenant
IDs, slugs, and cache components are bounded ASCII components, so separators,
whitespace, and oversized values cannot alter a storage key. Missing or invalid
scope fails closed with `ErrMissingScope`.

The package is an opt-in building block. It does not activate tenant routes,
migrations, RLS, workers, provisioning, billing, or shared-database mode in a
single-tenant build. Those boundaries require the ADR-0149 implementation
decision and their own dialect, isolation, capacity, and recovery evidence.

## Consequences

- Production code and contract tests share the same fail-closed scope semantics.
- Cache adapters can require a tenant scope before constructing keys.
- The primitive alone is not evidence of end-to-end tenant isolation or a SaaS
  readiness claim.
