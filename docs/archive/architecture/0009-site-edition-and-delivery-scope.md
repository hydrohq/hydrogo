# ADR-0009: Initial product, development order and delivery scope

- **Status:** Accepted
- **Date:** 2026-09-05
- **Related decisions:** [ADR-0001](0001-product-and-deployment-model.md), [ADR-0006](0006-extensible-database-adapters.md)
- **Implementation status:** Planned; no working product or certified adapter yet.

## Context

While the master plan described a news feed as the initial product, the backlog suggested a general Site Edition. The initial scope of the commercial product and the second module that verifies the architecture must be separated from each other. The tenancy decision was already accepted in ADR-0001; it is no longer an open question.

## Decision

The first sellable product will be **Site Edition**. The initial target customer is an agency client who wants corporate pages, services, team/reference content, and a contact form, delivered with a custom design. News is the second validation module; commerce is a separate product investment.

| Delivery | Scope | Exit condition |
|---|---|---|
| First working slice | Auth, Pages, limited Media, draft/publish, admin, public HTML and public JSON | The same use case works on two DBs; unauthorized access is rejected |
| Pilot Site Edition | Navigation, settings, SEO/redirects, revisions, manually defined limited contact form/inbox, backup/restore, customer theme | Actual customer work is completed without a core fork |
| SDK validation | News: review, scheduling, custom permissions; second Client Edition | Two different modules and two themes use the same extension boundary |
| Stable v1 | Certified SQLite/PostgreSQL, documented upgrade, support matrix, security/restore tests, verified CRUD generator | Release gates in the preparation document are passed |

The form inbox is not an arbitrary form designer. The initial form fields are defined in the edition code; it features size limit, spam/rate limit, retention, and personal data access. An advanced Form Builder is deferred.

Separate application, data, media, secrets, and backups are used per customer. Multi-tenant SaaS is not in the scope of v1. Locales in one deployment are not separate tenants. Edition identity provides a cache and operational namespace; it is not considered security isolation on its own.

## Schema and custom code

- Pages and News are first written manually. Once the common pattern is proven, the versioned collection definition used during build generates standard CRUD.
- The generator produces typed Go code, admin/API skeletons, repositories, and dialect migration skeletons **to be reviewed**. Automated DDL from the schema definition is not executed at runtime.
- Workflow, record/field authority, and destructive changes are not guessed. Generated access is initially rejected; permissions are explicitly defined.
- A new field requires a designated index, null/default, validation, and export behavior. Renaming is not guessed as drop+add; explicit field identity and rename records are required.
- The customer modifying content in the editor does not require a build. A new collection schema or Go/Templ customization requires a build and, if necessary, a migration. This boundary is specified in sales and developer documentation.

## Deployment scope

Compact single-server is the first pilot target. Standard/PostgreSQL is prepared for v1 with the same behaviors. Distributed architecture is preserved, but it is not sold as a certified option until multi-node operation, S3-compatible media, and failure tests pass. The completion of the first pilot does not mean Distributed support is finished.

The first production target is Linux amd64; the Windows development workflow is mandatorily verified. Linux arm64, Windows production, and other targets are included in the support matrix after separate build/driver/operation tests. The promise of a single binary refers to the application file generated per operating system and architecture; it does not mean the proxy, data, or external services are inside the binary.

## Alternatives and cost

A news-focused first product appeals to a narrower customer; a general Site Edition better measures agency repetition. Generating everything from schema forcefully generalizes workflows. Trying to finish all topologies from day one delays the pilot. The selected order does not eliminate subsequent certification work.

## Impact and validation

- SQLite/PostgreSQL contract tests start from the first Pages slice; PostgreSQL is not left to the end. New DBs are subject to ADR-0006 certification.
- Compact does not ask for an additional cache/queue service; resources are bounded. Distributed pool, worker, and backpressure validation is its own delivery gate.
- Pages public cache obeys locale, publication, and edition boundaries. Form/admin/preview do not enter the shared cache.
- Adding a module and schema changes enter the release manifest/migration plan; whether it is online or not is proven separately for the two dialects.
- Measure a sample job of the same scope with the current development method and with HydroGo: including setup, theme, collection, custom rule, test, build/deploy, and bug fixing. Report the initial design effort and the repeated setup time separately. Do not extract a 1–2 hour guarantee from a single successful example.
