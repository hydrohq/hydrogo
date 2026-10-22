# Decision records

| # | Decision | Status |
|---|---|---|
| [0001](0001-cms-focus.md) | Refocus HydroGo as a CMS | Accepted |
| [0002](0002-content-types.md) | One way to define a content type | Accepted |
| [0003](0003-social-sign-in.md) | Social sign-in for both admin staff and members | Accepted |
| [0004](0004-theme-appearance-css.md) | Theme contract carries the admin's Appearance CSS | Superseded by 0005 |
| [0005](0005-admin-svelte-spa.md) | Headless core, admin panel rebuilt as a Svelte SPA | Accepted |
| [0006](0006-freeze-legacy-peripherals-and-view-pipeline.md) | Freeze legacy peripherals and native view pipeline | Accepted |
| [0007](0007-template-cms-distribution.md) | Distribute HydroGo as a template CMS | Accepted |
| [0008](0008-business-application-capabilities.md) | Business-application capabilities in core | Proposed |
| [0009](0009-core-owned-event-outbox-and-migration-baseline.md) | Core-owned event outbox and a migration baseline for new databases | Accepted |

Write a record only when a change moves a module boundary, changes a public
contract (routes, API, config, migrations), or reverses a decision listed here.
Keep it under a page: context, decision, consequences. Older records live in
`docs/archive/architecture/` for history only.
