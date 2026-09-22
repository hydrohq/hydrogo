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

`unique`, `immutable`, and `default` remain metadata, not guaranteed runtime
behavior. Uniqueness needs explicit global-vs-tenant scope and portable
constraint/error mapping; immutability needs a trusted persisted baseline;
defaults need create-only/null/empty behavior and backward-compatible schema
rules. Do not claim these are enforced merely because the Blueprint carries
them.
