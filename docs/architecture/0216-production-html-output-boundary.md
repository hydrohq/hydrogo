# ADR-0216: Production HTML Output Boundary

- **Status:** Accepted
- **Date:** 2026-09-22
- **Decision:** Production-facing HTML must contain no HTML comments. Static
  theme/generated assets are linted before release, and the HTTP server applies
  a final `text/html` response guard that removes comments from server-rendered
  pages and HTMX partials. JSON, JavaScript, CSS, binary and encoded responses
  are not rewritten.

## Rationale

HTML comments can disclose implementation notes, generated-code guidance,
internal routes and unfinished work. Source comments in Go templates are not a
safe production boundary because they can become response bytes. The runtime
guard is therefore paired with source lint and render tests.

The guard buffers ordinary server-rendered HTML. Streaming or compressed
responses are not silently transformed; those response classes must not use
HTML comments and are covered by their own content contracts.
