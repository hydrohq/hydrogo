# Endpoint Observability Validation — 2026-09-22

The endpoint metrics contract now verifies normalized API routes, bounded
latency percentiles, status classes, rate-limit counts and request/response
bytes. The admin report table and CSV export expose the same bounded snapshot.

```text
go test ./internal/observability ./internal/ui ./cmd/hydro -count=1
ok  github.com/ardaozyildirim/hydrogo/internal/observability
ok  github.com/ardaozyildirim/hydrogo/internal/ui
ok  github.com/ardaozyildirim/hydrogo/cmd/hydro
```

The test sends `/api/v1/project/p-1` and confirms the report contains
`/api/v1/project/{id}` rather than the customer-controlled ID.
