# R5 request correlation validation — 2026-09-14

- `go test ./internal/observability` passes.
- HTTP logs include request, actor, and session IDs when present.
- Regression coverage confirms query strings, email addresses, token hashes,
  and other secret/PII values are not emitted.
- Job correlation is intentionally still open in the master task list.
