# R137 — Generated immutable field enforcement

**Date:** 2026-09-23  
**Status:** Implemented; local verification recorded below.

Generated global and tenant-scoped repository updates now place every declared
required scalar immutable value in the atomic `WHERE` predicate. A mismatch or
missing row returns `ErrResourceConflict`, which generated HTTP transports map
to a safe 409 response. Nullable immutable fields and many-valued immutable
relations are rejected at Blueprint validation because current models do not
preserve their required presence/baseline semantics.

No runtime DDL, migration, worker, or extra Core service is introduced.

## Verification

- `go test ./internal/generator -run 'TestGeneratedImmutableFieldsUseAtomicUpdateGuards|TestImmutableFieldRequiresRepresentableScalarValue|TestCoreCollectionFixturesGenerateDeterministically' -count=1` — passed. Generated repositories were executed against SQLite for global and tenant-scoped update/reject/update flows.
- `go test ./... -p 1 -timeout=5m` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed.
- Live PostgreSQL execution: not run; the generated SQL contract uses only
  placeholders, equality, `AND`, and affected-row inspection, but real-server
  verification remains required for release certification.
