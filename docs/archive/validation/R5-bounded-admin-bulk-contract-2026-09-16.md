# R5 bounded Admin bulk contract — 2026-09-16

Admin Studio now has a provider-neutral bulk request envelope with a default
500-record bound. Input is trimmed and sorted deterministically; malformed,
duplicate, control-character, empty, and over-limit selections fail closed.
Mutation and authorization remain module-owned.

Verification:

```text
go test ./internal/adminstudio -count=1
PASS
```
