# R5 opt-in Admin bulk list surface — 2026-09-16

Admin Studio lists now support an opt-in `BulkActions` metadata surface. Only
collections that declare actions receive selection controls; missing row IDs
fail closed and action URLs remain constrained to local paths.

Verification:

```text
go test ./internal/adminstudio ./internal/generator -count=1
PASS
```
