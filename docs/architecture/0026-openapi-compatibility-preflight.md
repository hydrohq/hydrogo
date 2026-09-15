# ADR-0026: OpenAPI compatibility preflight

- Status: Accepted
- Date: 2026-09-15

## Context

Generated and hand-authored clients need a deterministic check before a
published API removes a path, operation, response, or response property.
Database compatibility does not prove HTTP compatibility.

## Decision

`hydro validate api --baseline BASELINE --current CURRENT` compares two
OpenAPI JSON documents and fails closed when a previously published path,
method, response code, response media type, or response property disappears.
Additive changes remain allowed. The check is provider-neutral and does not
infer business semantics, authorization, or migration safety.

## Consequences

Client Editions can run the check in CI as a lightweight API gate. A breaking
change still requires a new API major/deprecation plan and an ADR; this command
is a preflight signal, not a replacement for reviewed contract tests.
