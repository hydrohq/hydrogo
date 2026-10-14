# ADR-0116: Make generated clients consume bounded queries

- **Status:** Accepted
- **Date:** 2026-09-16

The generated TypeScript client now imports its resource query helper instead
of creating an unbounded list URL. `list(query, signal)` validates through the
generated bounded query contract, encodes the stable query string, and passes
`AbortSignal` to `fetch`. The API remains source-compatible for callers that
invoke `list()` with no arguments.
