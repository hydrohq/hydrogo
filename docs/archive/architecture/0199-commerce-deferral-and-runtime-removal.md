# ADR-0199: Defer Commerce and remove its runtime surface

- **Status:** Accepted
- **Date:** 2026-09-21
- **Supersedes for the current release:** ADR-0017, ADR-0029, ADR-0031, ADR-0032, ADR-0034, ADR-0035, ADR-0036, ADR-0038, ADR-0174 and ADR-0175.
- **Related:** ADR-0009, ADR-0039.

## Decision

Commerce is not part of the current HydroGo product. Remove its Go module,
public/admin/API/GraphQL routes, payment-provider configuration, generated
presets, client contracts and product claims. New configurations reject the
`commerce` module name.

Existing Commerce migrations remain immutable historical records. They are not
selected for new non-Commerce profiles. Existing customer databases retain
their Commerce tables and data without a runtime route, worker, provider, or
background process accessing them. Any future Commerce return requires a new
ADR, a versioned compatibility/activation plan, explicit customer data
handling, and SQLite/PostgreSQL certification.

## Consequences

The supported product is Site Edition: Pages, News, Media, site services,
members and headless content APIs. Commerce-specific payment secrets are no
longer read or validated. Historical documentation and validation reports are
preserved as evidence, not as current product scope.
