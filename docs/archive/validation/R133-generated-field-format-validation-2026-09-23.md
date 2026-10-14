# R133 — Generated field format validation — 2026-09-23

## Scope

Generated validation now executes Blueprint `email`, `date`, `datetime`, and
`url` hints before Create/Update persistence. It rejects format hints on
non-text fields, counts min/max length by Unicode code point, and rejects
invalid UTF-8 when text length constraints are enabled. Only Go standard
library packages are emitted as imports.

## Verification

- `go test ./internal/generator -run 'Test(CoreCollectionFixturesGenerateDeterministically|GeneratedServiceBindsValidationContract|GeneratedValidationExecutesDeclaredFormats)' -count=1` — passed; generated package compiled and exercised valid and invalid format/Unicode cases.
- `go test ./... -p 1 -timeout=5m` — passed.
- `go vet ./...` — passed.
- `git diff --check` — passed.

## Deferred semantics

At the time of this validation, `unique`, `immutable`, and `default` remained
metadata. Uniqueness was subsequently specified and implemented by
[ADR-0258](../architecture/0258-tenant-scoped-blueprint-uniqueness.md) and
validated separately in [R135](R135-generated-unique-index-contract-2026-09-23.md).
Immutability still needs a trusted persisted baseline; defaults need create-only
field-presence semantics. Do not claim either is enforced merely because the
Blueprint carries it.
