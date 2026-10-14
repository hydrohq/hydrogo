# R111 — Generated route composition contract

Date: 2026-09-22

## Scope

The generator now emits `generated/<resource>/runtime.go` with a
policy-neutral `Register<Name>Route` helper. The helper composes the generated
repository, service and HTTP transport on `http.ServeMux`; it does not run DDL,
choose authorization, resolve tenants, install middleware, or start workers.

## Evidence

- `go test ./internal/generator -count=1` — PASS.
- `go test ./examples/article-edition/... ./examples/project-edition/... -count=1` — PASS;
  both example composition roots now call their generated registration helper.
- Generator determinism tests now require `runtime.go` and assert the named
  registration helper and policy guard are present.
- Tenant-scoped generation emits a resolver-required registration signature;
  collection/global generation emits the non-tenant signature.
- Generated `RUNTIME_WIRING.md` identifies the helper and preserves explicit
  migration, policy, CSRF, cache, capacity, and readiness review items.
- ADR index drift and HTML comment gates remain covered by the repository's
  local verification hook.

## Boundary result

This closes the repetitive route-composition step without making generated
Core code guess business policy. Existing constructors and transports remain
source-compatible, and inactive resources still have no runtime effect until
the Edition composition root calls the helper.

Production reverse-proxy, authenticated browser, tenant RLS, and capacity
certification remain external gates and are not claimed by this report.
