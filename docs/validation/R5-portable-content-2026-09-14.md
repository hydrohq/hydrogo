# R5 portable content validation — 2026-09-14

- `go test ./internal/portable` passes, including SQLite relation/revision/
  locale round trips and secret/session exclusion assertions.
- Decoder limits, unknown-field rejection, trailing-data rejection, and
  parent-before-revision insertion are covered.
- PostgreSQL round-trip evidence remains an external gate.
