# R135 — Generated unique index contract — 2026-09-23

## Scope

`validation.unique` now emits deterministic unique indexes. Indexes are
resource-global for non-tenant resources and `(tenant_id, field)` scoped for
tenant resources. Many-valued relation fields reject scalar uniqueness.
Migration metadata warns that pre-existing duplicates must be resolved before
operator application.

## Verification

- `go test ./internal/generator -run 'TestGeneratedUniqueIndexesAreTenantScopedAndPortable|TestCoreCollectionFixturesGenerateDeterministically' -count=1` — passed; generated migrations applied to SQLite, with global duplicate rejection and tenant-local uniqueness/cross-tenant reuse verified.
- `go test ./... -p 1 -timeout=5m` — passed.
- `go vet ./...` — passed.
- `scripts/verify-edition-footprint.ps1` — passed; compact Workspace 21,363,712 bytes / 152 non-standard packages, within 22 MiB / 160 package budgets.
- `git diff --check` — passed.

## Limits

No live PostgreSQL service was exercised by the local SQLite test. Stable
public error classification, duplicate cleanup plans for existing customer
data, RLS authorization, and deployment rollout remain edition/release gates.
