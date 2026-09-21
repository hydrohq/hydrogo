# ADR-0035: Secure payment return URL validation

- **Status:** Superseded by ADR-0199
- **Date:** 2026-09-12
- **Related decisions:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0017](0017-commerce-consistency-and-payments.md), [ADR-0032](0032-payment-provider-transport-safety.md)

## Decision

Configured Iyzico callback and PayTR success/failure URLs must be absolute HTTPS
URLs with a hostname. URLs containing credentials, fragments, or opaque forms
are rejected during configuration validation.

## Consequences

An insecure or malformed return path prevents the process from starting rather
than exposing payment state in a browser redirect. Query strings and distinct
paths remain supported for provider correlation. Local payment-provider testing
must use an HTTPS tunnel or a test provider that does not require these return
URLs; this is an intentional production-safety boundary and requires no schema
or runtime resource change.
