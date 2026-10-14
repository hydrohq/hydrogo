# ADR-0042: Referer redirect boundary for admin mutations

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0013](0013-identity-and-security-boundaries.md), [ADR-0035](0035-secure-payment-return-url-validation.md)

## Context

The member administration active-state form may return the operator to the
page that submitted the mutation. The HTTP `Referer` header is supplied by the
client and cannot be treated as a trusted navigation target. Redirecting it
verbatim creates an open-redirect primitive.

## Decision

Admin member mutation handlers may use `Referer` only after passing the shared
`security.SafeReturnPath` in-origin path validator. Absolute URLs, scheme
relative URLs, encoded traversal/normalization tricks, and control characters
fall back to the canonical member detail route. IDs used in that fallback are
path-escaped.

## Consequences

- Forged external `Referer` values cannot send an operator to an attacker site.
- Same-origin admin navigation remains convenient where the browser provides a
  valid path.
- The policy is centralized with login/member/payment return-path checks and
  does not depend on a proxy or browser security header.
