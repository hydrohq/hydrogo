# R5 validation: repository error mapping

## Local evidence

- `go test ./modules/members/sqlstore ./modules/members` passes.
- A real SQLite unique-email violation maps to `members.ErrMemberExists`.
- A PostgreSQL SQLSTATE `23505` fixture maps to the same domain error.
- Pages, commerce, and durable job conflict paths reuse the shared adapter
  classifier instead of parsing driver errors independently.
- Unrelated errors retain their identity for retry/observability decisions.

## Remaining gate

The PostgreSQL fixture does not replace execution against a real PostgreSQL
service. Multi-instance race behavior and production driver/version coverage
remain part of the PostgreSQL and capacity gates.
