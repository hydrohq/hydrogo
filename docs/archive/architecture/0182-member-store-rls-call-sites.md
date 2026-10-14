# ADR-0182: Bind member stores to tenant transactions

- Status: Accepted (member/session/audit stores covered; remaining direct pool paths open)
- Date: 2026-09-20
- Related decisions: [ADR-0173](0173-tenant-scoped-members-and-sessions.md), [ADR-0180](0180-postgres-tenant-rls-enforcement.md), [ADR-0181](0181-auth-identity-rls-call-sites.md)

## Decision

Member records, member sessions, and member audit listing now execute through
the tenant-aware transaction boundary. Row-producing operations use the
`database.WithinQuery` contract so PostgreSQL RLS binding and result scanning
share one physical transaction. The existing tenant predicates remain in
place as the first application-level defense.

## Consequences

- Member login/session validation, mutation, listing, and audit projection no
  longer issue unbound direct pool queries in shared mode.
- SQLite behavior remains unchanged apart from each operation having a short
  transaction boundary.
- Other handlers and stores that still accept raw `*sql.DB` require the next
  call-site audit and a live PostgreSQL isolation drill.
