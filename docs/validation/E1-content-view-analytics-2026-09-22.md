# Content View Analytics Validation — 2026-09-22

The analytics contract now supports a consent-gated `analytics_content_daily`
aggregate for tenant/content/locale dimensions. The asynchronous queue is
bounded and reports dropped/failed jobs without blocking a public request.

```text
go test ./extension/v1 ./internal/analytics/... ./tests/contract ./internal/database -count=1
ok  github.com/ardaozyildirim/hydrogo/extension/v1
ok  github.com/ardaozyildirim/hydrogo/internal/analytics
ok  github.com/ardaozyildirim/hydrogo/internal/analytics/sqlstore
ok  github.com/ardaozyildirim/hydrogo/tests/contract
ok  github.com/ardaozyildirim/hydrogo/internal/database
```

The store test confirms that two consented Article views become one aggregate
row with `view_count=2`, while the queue test confirms non-blocking bounded
enqueue and asynchronous delivery.
