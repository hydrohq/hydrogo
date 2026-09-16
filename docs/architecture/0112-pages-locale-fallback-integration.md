# ADR-0112: Configured locale fallback for public pages

- **Status:** Accepted
- **Date:** 2026-09-16

## Context

The locale fallback contract existed as a provider-neutral helper, but public
page reads still queried only the requested locale. That made a configured
language fallback unusable for pages and produced inconsistent behavior across
the HTML and JSON routes.

## Decision

`modules/pages/service.ContentService` performs an opt-in fallback chain when
`SupportedLocales` is configured: exact locale, primary language, then the
configured default. Each candidate uses its own bounded cache key. Fallback is
attempted only for `sql.ErrNoRows`; validation or infrastructure errors are
returned immediately. Empty configuration preserves exact-only behavior.

## Consequences

- HTML and JSON page routes share deterministic fallback behavior through the
  application service.
- Missing translations can be served without duplicating route handlers.
- Deployments must configure a validated locale list and default locale; no
  database or cache dialect details leak into the module contract.
- Product acceptance still needs real multilingual content across supported
  editions.
