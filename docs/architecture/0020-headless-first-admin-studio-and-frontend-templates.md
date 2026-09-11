# ADR-0020: Headless-first Admin Studio and frontend template boundary

- **Status:** Proposed
- **Date:** 2026-09-10
- **Related decisions:** [ADR-0009](0009-site-edition-and-delivery-scope.md), [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0015](0015-versioned-http-api.md), [ADR-0016](0016-structured-page-blocks-and-theme-slots.md), [ADR-0018](0018-filesystem-theme-bundles.md)

## Context

The current integrated presentation can carry the JSON API, server-rendered public view, and theme bundles within the same binary. This is fast for small Site Edition deliveries; however, for customers developing frontends, it blurs the separation between API and HTML routes and increases the cost of manually developing an admin screen for every module.

The product goal must satisfy two requirements simultaneously:

1. A secure admin CRUD surface should not be rewritten for standard data-driven modules.
2. The customer must be able to develop a frontend with their own framework and deployment without forking HydroCore.

## Proposed decision

The default product direction of HydroGo becomes **headless-first**:

- HydroCore provides domain services, versioned API, media, auth, authorization, workflow, audit, and commerce infrastructure.
- **Admin Studio** generates list, detail, form, relation, publish, and audit surfaces with metadata derived from the module spec.
- The public frontend is a separate client; it consumes the Core API. Core is not obligated to produce public HTML.
- Official frontend starters/templates are distributed as separate repositories or separate packages; they do not carry any business logic embedded in Core.
- The existing filesystem theme system is not removed; it remains as an optional `presentation=integrated` adapter for small integrated setups. A template does not replace the Core API contract.

Route ownership is strictly separated:

```text
/admin/...  Admin Studio and operator surface
/api/v1/... Versioned machine API
/webhooks/... Provider callbacks
frontend domain or separate app  Customer view
```

The `/public/...` namespace will not be expanded as a new frontend contract. Existing public HTML/API surfaces are preserved during the documented compatibility period or moved to a separate adapter with a versioned transition guide.

## Admin Studio contract

A collection/module spec explicitly defines at least the following:

- static field ID, type, required/null/default behavior, and validation;
- relations, indexes, locale/revision/export behavior;
- field/record authorization and workflow capabilities;
- admin visibility, sorting, filtering, and form widget hint;
- API exposure and projection definition;
- migration impact and rename/deprecation record.

The spec only generates standard CRUD metadata. Custom workflows, financial rules, provider integrations, and destructive transformations require explicit module code and reviewable migrations. DDL is not executed by loading a spec at runtime.

Admin Studio does not bypass Core services to directly access the DB. Every mutation goes through the same application service, authorization, CSRF, audit, and outbox policy.

## Frontend template model

Official templates are separate starters:

| Template | Scope | Core dependency |
|---|---|---|
| `site-starter` | SEO, navigation, pages, media, contact form | REST/GraphQL SDK |
| `storefront-starter` | catalog, variants, cart, checkout, account | Commerce API SDK |
| `editorial-starter` | news listing/detail, locale, preview integration | Content API SDK |

Each starter has its own lockfile, environment sample, e2e fixture, and Core API compatibility matrix record. Themes remain only as an asset/layout bundle for demo or integrated deployment; they are not a runtime dependency of the customer frontend.

## Transition and compatibility

- `/api/v1` is the main machine contract; ADR-0015 alias behavior is preserved.
- Before existing server-rendered routes are removed, user-visible links are migrated to frontend adapter URLs; the JSON endpoint and HTML endpoint are not mixed under the same route name.
- In the first phase, Admin Studio proves behavioral equivalence with existing handlers on Pages, News, and Commerce. Redirecting old admin routes to new Studio screens is done only after preserving the form/CSRF/audit contract.
- The new spec domain enters the migration plan, release metadata, and SQLite/PostgreSQL contract kit. Published migrations or field IDs are not retroactively changed.

## Acceptance gates

This proposal is only accepted if the following experiments succeed:

1. A deterministic Admin Studio surface is generated from the same spec for Pages and News; custom workflow code is preserved.
2. A separate storefront starter completes the login, cart, and checkout happy path via only `/api/v1` in an e2e test.
3. The integrated theme and headless edition consume the same published projection; no draft or private media leak occurs.
4. Generated admin mutations on SQLite and PostgreSQL pass the same authorization, conflict, audit, and migration contract tests.
5. In the Compact profile, the headless edition does not load public theme assets/routes/workers.
6. The old integrated edition passes compatibility fixtures with the new Core version, and the new starter with the old supported API version.

## Alternatives

- **Theme inside Core only:** provides a fast start but weakens frontend freedom and deployment separation.
- **Headless only, no starters:** keeps the architecture clean but increases the initial customer cost.
- **Runtime schema builder:** appears fast; creates an unauditable risk for migrations, authorization, audit, rollback, and DB portability.

The proposed hybrid model simultaneously preserves the automated admin experience and frontend freedom; however, a stable v1 contract is not established without validation in two real customer flows.
