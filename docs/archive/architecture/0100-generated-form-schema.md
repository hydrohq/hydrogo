# ADR-0100: Generated form schema artifact

**Status:** Accepted  
**Date:** 2026-09-16

## Context

The form contract existed as a reusable runtime boundary, but generated
resources still required a separate hand-written form metadata description.
That weakens the single-spec workflow needed for fast prototypes.

## Decision

The collection generator emits a deterministic `form.schema.json` alongside
the HTML form. It contains the collection ID/title and bounded field metadata
(type, label, required). Types are derived only from explicit field type or
presentation metadata; authorization, validation beyond the schema, consent,
and inbox delivery remain explicit review/module work.

## Consequences

Client Editions can feed the same artifact into a form renderer without adding
Core dependencies. Existing generated files remain compatible except that the
new artifact is now required by the compatibility checker.

## Verification

Generator tests and preset smoke verify the artifact is generated and included
in compatibility validation.
