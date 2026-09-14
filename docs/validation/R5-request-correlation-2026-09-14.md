# R5 request correlation validation — 2026-09-14

- `go test ./internal/observability` passes.
- HTTP logs include request, actor, session, and job IDs when present; news and
  webhook workers propagate only opaque job/event IDs.
- Regression coverage confirms query strings, email addresses, token hashes,
  and other secret/PII values are not emitted.
