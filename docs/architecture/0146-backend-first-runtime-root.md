# ADR 0146: Backend-first runtime root and optional demo frontend

- Status: Accepted
- Date: 2026-09-18
- Supersedes: [ADR-0090](0090-configured-public-home-locale.md) for the default root fallback

## Context

HydroGo is currently being developed as a backend-first platform. Redirecting
`/` to seeded marketing pages made the runtime appear frontend-led, introduced
unnecessary visual-test scope, and coupled a production root route to local
demo data. The demo seed is already explicit (`seed-demo --confirm-demo`) and
should remain available for local browser validation without becoming the
default product surface.

## Decision

The default `GET /` route serves a small technical runtime status page with
links to health, readiness, API, and Admin Studio endpoints. A validated public
`site.home_path` continues to override this behavior when an installation has
explicitly configured a product frontend. The seeded demo remains available at
the locale-aware `/public/pages/{locale}/demo/html` path after an explicit
`seed-demo --confirm-demo` invocation, but it is linked as optional demo content
and is never the default fallback.

The Pages module is not required for the technical root route. No demo assets,
pages, news records, products, or member accounts are created during normal
runtime startup; those remain owned by the explicit demo seed command.

## Consequences

- Backend/API deployments have a truthful, low-maintenance root surface.
- Frontend visual regression work is opt-in and can be scheduled as a separate
  product phase.
- Existing configured public home paths remain compatible and validated.
- Local demo workflows keep their existing URLs and seed behavior.
- No migration or cache contract changes are required.
