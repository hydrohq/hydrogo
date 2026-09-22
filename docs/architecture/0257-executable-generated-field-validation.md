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

Generated Admin forms map `email`, `date`, and `url` to corresponding HTML
input types and render typed `minlength`/`maxlength` hints for input and
textarea components. These browser constraints improve editing feedback only;
they are not accepted as a substitute for generated server validation.

This uses no runtime service or external dependency and runs before generated
service Create/Update repository writes. Validation errors remain generic
field-level messages and do not echo user input.

This decision does not define uniqueness scope or create-only default
application. Uniqueness is defined by ADR-0258. Supported scalar immutable
fields are enforced atomically under ADR-0260; nullable fields and many-valued
relations remain rejected until their presence/baseline semantics are modeled.
Create defaults remain Edition-owned until a versioned field-presence contract
is adopted.

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
