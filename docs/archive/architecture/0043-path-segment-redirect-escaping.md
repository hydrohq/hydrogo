# ADR-0043: Escape dynamic redirect path segments

- **Status:** Accepted
- **Date:** 2026-09-14
- **Related decisions:** [ADR-0015](0015-versioned-http-api.md), [ADR-0042](0042-referer-redirect-boundary.md)

## Context

Admin mutation handlers construct local redirect locations using resource IDs
from request paths or persistence. HTML escaping and query escaping do not
provide the correct semantics for a URL path segment, and raw concatenation
allows separators or control-like input to alter the target path.

## Decision

Every dynamic identifier inserted into an admin redirect path is encoded with
`url.PathEscape`. Redirects remain fixed-origin paths; IDs are not trusted to
provide additional path segments.

## Consequences

- Redirect locations preserve path-segment boundaries for malformed or hostile
  IDs while retaining ordinary UUID behavior.
- HTML and query escaping remain reserved for their respective output contexts.
- The transport layer no longer relies solely on repository-level ID validation.
