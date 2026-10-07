# ADR-0353: Public shared Admin Studio shell for generated Edition panels

- **Status:** Accepted
- **Date:** 2026-10-07
- **Supersedes:** None
- **Superseded by:** None
- **Related decisions:** [ADR-0210](0210-blueprint-scaffold-boundary.md), [ADR-0240](0240-generated-route-composition.md), [ADR-0346](0346-blueprint-admin-studio-workflows.md), [ADR-0347](0347-versioned-public-generated-admin-runtime.md), [ADR-0349](0349-explicit-generated-resource-activation.md)

## Context

Generated Admin routes already use the versioned `extension/v1/adminstudio`
runtime, but each Edition had to supply its own page renderer. The in-repository
Project Edition example imported `internal/ui`; a separate Go module cannot
import that package. A minimal custom renderer compiled externally, but it did
not provide the same Studio shell, typography, theme controls, or navigation.

Editions need a public default renderer that gives generated resource panels
the shared HydroGo Studio experience without moving authentication, policy,
tenant scope, CSRF, API, or editorial decisions into Core.

## Decision

- `extension/v1/adminstudio` exposes `StudioPageRenderer` and a typed
  `PageIdentity` containing the authenticated user's ID, email, and Admin or
  Editor role for presentation.
- The renderer composes the existing Core Admin Studio shell and typed Templ
  body. It is an additive v1 extension contract and adds no parallel theme or
  browser runtime.
- The identity resolver supplies presentation context only. The generated
  handler continues to require Edition-owned `Authorize` and
  `CanReadResource` callbacks; CSRF, API, tenant, workflow, and persistence
  boundaries remain Edition-owned.
- Generated resource links continue to come from the explicitly selected,
  permission-filtered registry. The renderer does not activate resources or
  invent navigation policy.
- Editions may keep a custom `RenderPage` callback when they need a different
  shell. View-only or custom roles also provide their own renderer; the shared
  helper accepts only the existing Admin and Editor shell roles.

## Alternatives

- Keep requiring every Edition to provide its own shell: rejected because it
  duplicates work and leaves generated panels visually inconsistent.
- Import `internal/ui` directly: rejected because Go forbids it outside the
  HydroGo module and it is not a versioned customer contract.
- Generate a copy of the shell into each Edition: rejected because it would
  duplicate UI/security fixes and drift from the shared Admin Studio.
- Let presentation identity grant access: rejected because page identity is
  not authorization and Blueprint metadata is not policy.

## Consequences

### Benefits

- Independently compiled Editions can use the same Admin shell and generated
  navigation through the supported public contract.
- The existing HydroGo product owns shared visual and accessibility fixes;
  Editions retain control over identity resolution and policy.
- External-module compilation tests now exercise the same shell used in the
  Project Edition browser acceptance flow.

### Costs and risks

- The exported identity and renderer signatures are v1 compatibility promises.
- The helper supports Admin and Editor presentation roles; Editions with
  view-only/custom roles must render an appropriate shell themselves.

## Scale impact

- **Compact:** uses the already embedded server-rendered shell. It adds no
  service, worker, route, asset, browser runtime, or database resource.
- **Standard:** uses the Edition's existing handler, middleware, and pool.
- **Distributed:** remains stateless; the Edition resolves identity and tenant
  scope per request.

## Database impact

- **SQLite:** no schema, query, transaction, or migration changes.
- **PostgreSQL:** no schema, query, transaction, or migration changes.
- **Future adapters:** no database behavior is introduced.

No migration is required; there is no persisted state and no rollback DDL.

## Cache impact

No cache, cache key, TTL, invalidation, stampede, or fallback behavior changes.
Generated Admin responses keep their existing private/no-store behavior.

## Deployment and compatibility

The public renderer is additive to the versioned v1 package. Editions rebuild
against a compatible runtime; old binaries and persisted data are unaffected.
There are no migrations, shared mutable state, workers, or assets to roll back.

## Validation

- `go test ./extension/v1/adminstudio` verifies role validation and the public
  contract.
- `go test -mod=mod ./...` in `tests/external-edition` compiles and renders the
  shared shell from an independent module without importing `internal/ui`.
- `go test ./examples/project-edition/...` verifies the example uses only the
  versioned renderer contract.
- Chromium Admin CRUD E2E verifies shared shell/navigation, Turkish and English
  locale consistency, mobile overflow, CRUD, and axe-core WCAG 2.1 A/AA checks.
- Repository Go tests and the required module/import/migration guards remain
  mandatory.

## References

- [Admin Studio validation](../validation/generated-admin-public-studio-renderer-2026-10-07.md)
- `extension/v1/adminstudio`
- `tests/external-edition`
