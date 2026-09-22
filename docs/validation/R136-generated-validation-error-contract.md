# R136 — Generated validation error contract

**Date:** 2026-09-23  
**Status:** Implemented; local verification recorded below.

Generated non-tenant and tenant resource transports now map typed Blueprint
field validation failures to a generic JSON HTTP 422 response with stable
field/code pairs. Only errors wrapping the generated `ErrResourceConflict`
sentinel return generic 409; unclassified service and database failures return
generic 500 without exposing underlying errors. Correlation IDs are echoed
only when they match a bounded ASCII allowlist.

The generated OpenAPI describes collection create/list and item read/update/
delete operations, requires the `{id}` path parameter on item operations, and
documents validation and conflict responses. The generated fixture invokes a
real handler with invalid email input and checks that neither the invalid
value nor internal parser details appear in the response.

## Verification

- `go test ./internal/generator -run 'TestCoreCollectionFixturesGenerateDeterministically|TestGeneratedServiceBindsValidationContract|TestGeneratedValidationExecutesDeclaredFormats|TestGeneratedTransportReturnsSafeFieldValidationError|TestGeneratedTransportDoesNotMisclassifyRepositoryErrors' -count=1` — passed.
- `go test ./... -p 1 -timeout=5m` — passed on Windows; SQLite/PostgreSQL contract tests passed as included in the suite (this does not imply every provider-backed test ran against a live PostgreSQL instance).
- `go vet ./...` — passed.
- `git diff --check` — passed.

This evidence covers the generated transport fixture and OpenAPI output. It is
not a live PostgreSQL or deployed Edition certification.
