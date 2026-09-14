# ADR-0090: Derive the public home locale from runtime configuration

## Status

Accepted

## Context

The public root route redirects to the seeded home page. Hard-coding `tr` made
the root route incorrect for installations whose configured default locale is
different.

## Decision

The composition root derives the locale segment from `I18N.DefaultLocale`,
falls back to `en` when it is empty, and escapes the segment before constructing
the route. The seeded `demo` slug remains intentionally local-demo data and is
not presented as a customer home-page contract.

## Consequences

Configured multilingual installations now land on their default locale. The
customer-facing home slug still requires a site-settings/content-selection
decision before it can replace the demo seed safely.
