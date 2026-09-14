# ADR-0086: Keep member browser transport errors generic

## Status

Accepted

## Context

Member authentication pages run in an untrusted browser. Network and API
exceptions can contain implementation details, proxy text, or service metadata.
Displaying runtime error details directly makes them part of the user-facing
contract and can expose information that is useful to an attacker but not to a
member.

## Decision

Member response sanitization removes runtime error expressions from rendered
authentication pages and leaves a stable generic transport message. Diagnostic
details remain server-side and request correlation is the operational source for
investigation.

## Consequences

The browser experience is predictable and does not disclose runtime internals.
Support and operators must use correlated server diagnostics when investigating
failures. A regression test rejects raw runtime details and legacy browser
identity storage markers.

## Validation

See [R5 user-facing error redaction](../validation/R5-user-facing-error-redaction.md).
