# ADR-0135: Role-specific editorial transition policy

- **Status:** Accepted
- **Date:** 2026-09-16
- **Related decisions:** [ADR-0129](0129-news-editorial-transition-policy.md), [ADR-0013](0013-identity-and-security-boundaries.md)

## Decision

The versioned editorial workflow contract supports an optional bounded
`RoleTransitions` allowlist. A role with an explicit policy may perform only
the listed transitions; roles without a policy retain the general graph. The
News service resolves the current role from request context and enforces the
policy inside the same transaction as the state mutation. Presentation may
use the same contract to hide unavailable actions, but it is never the source
of authorization.

Role names and state/transition sets are bounded and validated. Persistence of
arbitrary custom states and an Admin workflow editor remain separate work;
this policy does not weaken the existing status storage contract.

## Evidence

`extension/v1/workflow_test.go` covers policy validation and fallback behavior;
`modules/news/workflow_test.go` proves an editor cannot publish when the role
allowlist denies it. The composition root resolves the authenticated role
without exposing auth internals to the extension contract.
