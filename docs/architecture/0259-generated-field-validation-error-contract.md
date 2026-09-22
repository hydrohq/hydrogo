# ADR-0259: Generated field-validation error contract

- **Status:** Accepted
- **Date:** 2026-09-23
- **Related:** ADR-0257, ADR-0258

## Context

Generated resource validators execute Blueprint constraints, but returning
implementation errors or submitted values from generated transports would leak
details and leave API consumers unable to associate a failure with a field.
The generated OpenAPI document also needs to describe the actual CRUD surface
and the validation response used by those transports.

## Decision

Generated validators return a typed `FieldValidationError` carrying only the
Blueprint field name and a stable validation code. Generated Create and Update
transports map that error to HTTP 422 and a bounded JSON envelope containing
`code`, a generic `message`, `fields`, and `request_id`. Only errors wrapping
the generated `ErrResourceConflict` sentinel map to HTTP 409; unknown service
or database errors map to generic HTTP 500. Their underlying database or
implementation text is never serialized. The optional request ID is echoed
only when it is at most 64 ASCII alphanumeric, dot, underscore, or hyphen
characters; otherwise it is omitted as an empty value.

Generated OpenAPI now describes collection GET/POST and item GET/PUT/PATCH/
DELETE operations, including the required `{id}` path parameter and the 409/422
mutation responses. This is a transport contract, not a promise that every
repository error is a conflict: Edition repository adapters must wrap known
domain conflicts with the sentinel before exposing this status.

## Consequences

- API clients can render field-level validation without receiving user input
  or internal errors.
- Generated transports remain generic; authorization and domain policy remain
  in Edition-owned policies/services.
- The 409 mapping is opt-in through an explicit sentinel; unclassified errors
  retain server-failure semantics.
- Runtime and OpenAPI contract tests must evolve together.

## Verification

The generator fixture executes the generated handler with invalid email input,
asserts the 422 field code and safe request ID, and verifies the submitted value
and implementation details are absent. Generator tests inspect the CRUD paths
and validation response in generated OpenAPI.
