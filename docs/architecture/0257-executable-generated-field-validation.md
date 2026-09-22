# ADR-0257: Execute generated string format validation

- **Status:** Accepted
- **Date:** 2026-09-23
- **Related decisions:** [ADR-0226](0226-versioned-field-validation-contract.md), [ADR-0210](0210-blueprint-scaffold-boundary.md)

## Context

ADR-0226 allows provider-neutral string format hints and states that generated
mutation services bind field validation. The generator preserved `format` in
the Spec but did not execute it in `Validate()`. It also measured string
length in bytes, which made the configured length differ from user-visible
text length for non-ASCII input.

## Decision

Generated `Validate()` enforces `email`, `date`, `datetime`, and `url` hints
using Go standard-library parsers. Email values must be a bare mailbox without
a display name; dates use `YYYY-MM-DD`; datetimes use RFC3339; URLs must be
absolute HTTP or HTTPS URLs with a host. Format hints are valid only on
`string` and `text` fields. String min/max length uses Unicode code-point
counting and rejects invalid UTF-8 when a length rule is configured.

This uses no runtime service or external dependency and runs before generated
service Create/Update repository writes. Validation errors remain generic
field-level messages and do not echo user input.

This decision does not define uniqueness scope, update-baseline semantics for
immutability, or create-only default application. Those metadata properties
remain Edition-owned until a versioned contract states tenant and lifecycle
semantics; generated code must not imply they are enforced automatically.

## Consequences

- Generated services now enforce all supported `format` hints consistently
  for API and non-Admin callers.
- Unicode text length is stable across UTF-8 storage rather than byte-count
  dependent.
- HTML validation remains advisory; server-side generated validation is the
  enforcement boundary.

## Verification

Generator tests assert emitted standard-library validators and reject format
hints on non-text fields. A generated fixture with all four validators is
compiled and executes valid/invalid email, date, datetime, URL and Unicode
length cases.
