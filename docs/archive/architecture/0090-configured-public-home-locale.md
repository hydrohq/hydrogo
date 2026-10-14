# ADR-0090: Derive the public home locale from runtime configuration

## Status

Superseded by [ADR-0146](0146-backend-first-runtime-root.md)

## Context

The public root route previously redirected to the seeded home page. Hard-coding `tr` made
the root route incorrect for installations whose configured default locale is
different.

## Decision

This decision remains valid for an explicitly configured `site.home_path`, but
its seeded-demo fallback is superseded. The default root no longer redirects to
demo content; see ADR-0146.

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

Newsroom header/footer navigation rewrites its legacy seeded `tr` links to the
resolved locale at render time, preventing locale-specific installations from
showing dead navigation targets.
