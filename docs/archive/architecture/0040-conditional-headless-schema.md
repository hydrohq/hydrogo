# ADR-0040: Conditional headless schema surface

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md), [ADR-0033](0033-runtime-module-resource-gating.md), [ADR-0039](0039-conditional-cross-module-migrations.md)

## Context

The public GraphQL endpoint is shared by Pages and Commerce. A route can be
enabled when either module is active, but exposing a resolver for an inactive
module leaks an unsupported capability and can query a table that the selected
profile deliberately did not migrate. The bootstrap also must pass the active
Commerce capability into the handler so an enabled catalog is not silently
disabled.

## Decision

The GraphQL schema is built from the active module set. The `page` root field
exists only when Pages is enabled, and `products` exists only when Commerce is
enabled. Bootstrap passes both capabilities explicitly; inactive fields are
absent from introspection and validation rather than returning a misleading
empty result. The endpoint remains read-only and keeps its existing byte,
complexity, and rate limits.

## Consequences

- Compact profiles do not expose or execute resolvers for inactive modules.
- Clients must feature-detect the schema or use the edition manifest before
  issuing module-specific queries.
- Enabling a module remains a deployment/configuration decision; this ADR does
  not certify PostgreSQL or production cross-origin browser behavior.

## Verification

`modules/api/graphql/handler_test.go` verifies field presence for Pages-only and
Commerce-only schemas, while the CLI wiring is covered by the full test suite
and generator/health smoke. Production browser and PostgreSQL certification
remain tracked in the open-gates register.
