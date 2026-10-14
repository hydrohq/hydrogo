# R110 — Tenant-scoped members and sessions

Date: 2026-09-19

## Evidence

Member and member-session stores accept a fail-closed tenant mode. Member
create, lookup, search, stats, status changes and session lifecycle operations
carry the validated scope. A SQLite test proves a scope-less lookup is rejected.

Commands:

- `go test ./modules/members/... ./cmd/hydro ./migrations -count=1`
- `go run ./tools/migrationguard`

Result: passed.

Mobile device-session store methods, auth user/session storage, identity
provisioning and production PostgreSQL certification remain open.
