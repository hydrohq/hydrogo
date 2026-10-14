# ADR-0001: Product and customer deployment model

- **Status:** Accepted
- **Date:** 2026-09-05

## Context

In customer projects based on PayloadCMS and PocketBase, forking or directly modifying the core made updates difficult. HydroGo should be both a marketable product and a platform that accelerates the agency's repeated CMS developments.

## Decision

HydroGo is positioned as a self-hosted and developer-focused **CMS product factory**.

- `HydroGo`: The product family and experience presented to the customer.
- `HydroCore`: Versioned engine independent of customer code.
- `Hydro Modules`: Versioned capabilities such as Pages, News, Media, Forms, SEO, and similar.
- `Hydro CLI`: Project, schema, build, upgrade, backup, and deployment tools.
- `Client Edition`: Brand, theme, enabled modules, content schemas, and custom business rules.

By default, each customer works with a separate application, database, media space, secrets, and backups. Even though multiple low-traffic customers can be hosted on the same machine, process and data isolation is maintained.

Customer requirements are not implemented by forking HydroCore. A new need:

1. Is solved with an existing module or contracted extension point.
2. If it is truly a general capability, it is converted into versioned Core/Module development.
3. If it is specific to the customer, it stays within the `Client Edition`.

The name of this approach is the **Zero Core Fork** rule.

## Deployment models

- **Managed:** The recommended commercial model operated by the agency.
- **Self-hosted:** The signed release package and operational documents are delivered to the customer.
- **Container:** An optional deployment format for enterprise infrastructures; it is not the mandatory operational mode of the product.

## Consequences

- Core updates can be developed centrally.
- Customer code remains small and understandable.
- Each customer pins their own Core and module versions.
- A build matrix, contract tests, and fleet inventory are required for a large number of customers.
- Full structural Templ overrides require a rebuild per customer; theme tokens are preferred for simple brand changes.

## Rejected alternatives

- HydroCore fork per customer
- Multi-tenant SaaS in the first release
- Loading untrusted third-party Go plugins at runtime
- Making Kubernetes the default deployment
