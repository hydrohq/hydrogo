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
the route. When the Site module is active, a public `site.home_path` setting may
override the fallback with a validated internal path. The seeded `demo` slug
remains intentionally local-demo data and is not presented as a customer
home-page contract.

## Consequences

Configured multilingual installations now land on their default locale, and a
site administrator can choose a safe internal home path without changing the
composition root. The local `seed-demo` command writes pages and news using the
same configured locale, so its fallback route remains resolvable. Missing,
private, or external values safely fall back to the configured locale/demo
route.
