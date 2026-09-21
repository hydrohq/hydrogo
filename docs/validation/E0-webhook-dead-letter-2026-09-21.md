# E0 Webhook Dead-Letter Validation — 2026-09-21

Evidence for ADR-0205:

- `internal/webhooks/worker_test.go` proves a delivery failure at the configured attempt threshold calls the dead-letter boundary and does not schedule another retry.
- `internal/outbox/sqlstore/store_test.go` proves a leased event is removed from `outbox_events` and retained with its failure reason in `outbox_dead_letters`.
- `migrations/202609210002_outbox_dead_letters` provides SQLite/PostgreSQL additive schema and migration metadata.

Executed checks:

```text
go test ./internal/outbox/sqlstore ./internal/webhooks ./internal/config ./internal/eventcatalog ./cmd/hydro -count=1
```

The live PostgreSQL path remains a deployment certification gate and requires `HYDRO_TEST_POSTGRES_URL`.
