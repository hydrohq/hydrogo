# R5 bounded async relation contract — 2026-09-16

Admin Studio now defines a provider-neutral async relation lookup envelope with
bounded collection/query/cursor/limit fields and a deterministic ID/label
option projection. Provider lookup, authorization and cursor handling remain
outside Core.

Verification:

```text
go test ./internal/adminstudio -count=1
PASS
```
