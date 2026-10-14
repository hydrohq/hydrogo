# ADR-0027: Provider-neutral field registry

- Status: Accepted
- Date: 2026-09-15

## Context

Payload-like velocity requires one field declaration to drive persistence,
validation, API schemas and admin widgets. Binding the schema to a single UI
provider would violate HydroGo's Client Edition and provider-neutral design
boundaries.

## Decision

The Admin Studio exposes a small provider-neutral `FieldRegistry` vocabulary
(text, number, boolean, date, media, rich text, select, relation, array and
blocks). Generators map schema types into this vocabulary. Client Editions map
the vocabulary to HTML, HTMX, React, shadcn or another design system; unknown
kinds fail closed.

## Consequences

New field kinds require validation, UI/API mappings and contract tests before
they become shared Core vocabulary. Provider packages remain optional and
inactive modules do not acquire their assets or runtime dependencies.
