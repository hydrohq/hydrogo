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
| [0012](0012-commerce-catalog-as-content-types.md) | Commerce catalog as content types, not a new engine feature | Accepted (Phase 1 of 4) |
| [0013](0013-commerce-cart-checkout-module.md) | `modules/commerce` for stock, cart, checkout and orders | Accepted (Phase 2 of 4) |
| [0014](0014-commerce-paytr-provider.md) | PayTR as a `commerce.PaymentProvider` | Accepted (Phase 3 of 4) |
| [0015](0015-commerce-starter.md) | `examples/commerce-starter` | Accepted (Phase 4 of 4) |
| [0016](0016-commerce-starter-catalog-sections.md) | Commerce starter catalog sections (category pages, search, galleries) | Accepted |
| [0017](0017-commerce-tax-shipping-discounts-currencies.md) | Tax, shipping, discount codes and display currencies | Accepted |
| [0018](0018-commerce-reconciliation.md) | Commerce reconciliation and drift alerting | Accepted |
| [0019](0019-commerce-order-notifications-and-member-login.md) | Order notifications and storefront member login | Accepted |
| [0020](0020-commerce-shipping-provider.md) | Generic `ShippingProvider`, Yurtiçi Kargo first | Accepted |

Write a record only when a change moves a module boundary, changes a public
contract (routes, API, config, migrations), or reverses a decision listed here.
Keep it under a page: context, decision, consequences. Older records live in
`docs/archive/architecture/` for history only.
