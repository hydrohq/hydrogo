# Headless-first Admin Studio and frontend starter plan

**Status:** Implementation experiments are ongoing; the architectural direction in [ADR-0020](architecture/0020-headless-first-admin-studio-and-frontend-templates.md) is still in the proposed state and will only be accepted after experiment gates.

## Target experience

A customer should be able to choose one of these two paths:

```text
hydro edition create --presentation=headless
  → Admin Studio + API
  → customer connects their own Next.js/Nuxt/Astro frontend

hydro frontend create storefront
  → separate frontend starter repository
  → API URL and public key/config are entered
```

In small setups, the `presentation=integrated` option continues with a server-rendered template; this option will not be the default frontend development model.

## Phase 0 — Usage research and boundary validation

**Goal:** Separate the repetition to be automated by the generator from the actual flow.

- Take the existing admin handlers/forms of Pages, News, and Commerce as inventory with fields, relations, workflows, and custom rules.
- Choose two reference business cases: corporate site + storefront with variants.
- Measure the user journey: adding a collection, adding a field, publish, catalog management, list/detail/cart on the frontend.
- Classify each URL as owned by admin, API, or frontend; record routes where API/HTML mixes.

**Output:** Repeated CRUD areas and rules requiring custom code are separated with written evidence. This phase does not freeze `module-spec` v1. Initial inventory: [Admin Studio surface inventory](validation/admin-studio-inventory-2026-09-10.md).

## Phase 1 — Module spec v0 and metadata compiler

**Goal:** Expand the existing generator to Admin Studio metadata.

- Design a versioned collection spec definition under `module-specs/`.
- Cover Field ID, validation, relation, permission, API projection, admin widget, and migration intent.
- Produce deterministic compiler output: Go typed metadata + Admin Studio manifest + reviewable migration draft.
- Preserve the separation of `generated/` and `custom/` with ADR-0010 rules.
- Do not produce runtime DDL, implicit rename/drop, or public API exposure without explicit permission.

**Acceptance:** Zero diff when the same spec is generated twice; custom files are unchanged; invalid/destructive specs are fail-closed rejected.

## Phase 2 — Admin Studio core

**Goal:** Produce an administrative surface that is consistent with metadata and secure.

- Common layout, navigation, list/detail/form screens, pagination, filter, relation picker, and validation error UI.
- Mutations only pass through the application service; there are no DB query or authorization shortcuts.
- Unify CSRF, CSP, optimistic-lock conflict, audit request ID, and field-level permission visibility.
- Migrate Pages and News to the Studio as the first two validation modules; keep special publish/schedule behaviors as custom panel extensions.

**Acceptance:** Old and Studio admin flows yield the same error code, audit, and state transition results in SQLite/PostgreSQL HTTP contract tests.

## Phase 3 — API client and frontend contract

**Goal:** Developing the frontend independently of the Core UI but connected to the Core.

- Generate a typed TypeScript client from an OpenAPI/GraphQL schema or write a versioned thin client.
- Document clear `/api/v1` contracts for public read, preview, auth/session, cart, and checkout, and add fixtures.
- Validate CORS, cookie domain, CSRF, and preview token decisions with [ADR-0021](architecture/0021-headless-browser-session-boundary.md).
- Give public route ownership to the frontend application; Core only serves API/admin/webhook paths.

**Acceptance:** A fixture frontend in another repository builds and runs e2e tests only with the SDK without importing Core internals.

## Phase 4 — Official frontend starters

**Goal:** Reduce headless startup cost.

1. `site-starter`: page list/detail, SEO, navigation, locale, media, and form.
2. `storefront-starter`: product, variant, authenticated cart, checkout, and provider redirect handling.
3. `editorial-starter`: news list/detail and published content cache/ISR strategy.

Starters are separate deployments; they do not add assets or Node runtimes to the Core binary. Each starter carries a Core version matrix, `.env.example`, seed fixture, and Playwright/e2e happy path.

**Acceptance:** In an environment seeded from scratch, a developer boots up the starter with the documented flow; login-cart-checkout and publish-to-rendered-page e2e evidence passes.

## Phase 5 — Positioning the integrated template as an adapter

**Goal:** Simplify the main presentation model while protecting existing theme investments.

- Restrict theme packages to the `presentation=integrated` option.
- Remove URL conflicts between public HTML routes and their API counterparts; write a deprecation/redirect plan for legacy paths.
- Maintain theme asset manifest, retained asset, and blue/green contracts.
- Test that theme routes, assets, and render dependencies are never loaded in a headless build.

**Acceptance:** The same content projection is displayed correctly, authorized, and cache-safely with different UIs by the integrated template and a separate starter.

## Phase 6 — Transition, support, and release

- Integrated → headless migration guide and compatibility matrix for existing editions.
- API deprecation windows, starter semver policy, and edition/starter lockfile compatibility.
- SQLite/PostgreSQL migration, rollback, blue/green asset, and API contract tests.
- Measuring duration and error rates with two real customer jobs; do not make the new spec surface v1 before the net benefit of automation is verified.

## Implementation order

First validate **admin automation** with Phases 0–2, then the **headless frontend experience** with Phases 3–4. Phase 5 is only done after starters are validated with real usage. This proves the new default experience without breaking existing integrated customers.

## Open decisions

- Typed TypeScript SDK code generation, or a minimal fetch client?
- Same-site cookie for frontend auth, BFF, or a token-based separate mobile API?
- Which signed-token and cache-bypass contract will the preview flow work with?
- How will the release/version compatibility matrix of the starters be linked to the Core release manifest?

These questions do not block starting the implementation, but they block freezing the external API and security surface of Phase 3.
