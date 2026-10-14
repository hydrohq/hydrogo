# R4 News schedule validation

**Date:** 2026-09-08  
**Status:** Initial News domain/repository, JSON/public route, and bounded worker lease slice succeeded; full admin UI and durable job backend gates remain open.

Migration `202609080012_news_items` adds revision, optimistic version, published revision, schedule generation, and time fields. The `modules/news` service exposes draft/update, immediate publish, retract, and generation-guarded scheduled publish flows, bindable to audit/outbox writers within the same transaction.

`internal/jobs.Queue` provides bounded/idempotent enqueue, lease expiry reclaim, and worker token validated ack/nack behavior. The News worker does not ack and republish a stale generation job; it nacks for retry on transient errors.

The contract test proves the following on SQLite:

- a stale editor update is rejected;
- publishing is rejected if the schedule generation is incorrect;
- the correct generation is published when due;
- it can be read via the published projection route;
- the public route cannot be found post-retract.

```text
go test ./tests/contract -run NewsContract -count=1
go test ./...
```

The PostgreSQL contract test is skipped if `HYDRO_TEST_POSTGRES_URL` is absent. JSON/public routes use the same service/repository projection; worker lease behavior is verified with an in-process contract. The durable job backend, full admin UI, and second Client Edition are subsequent R4 gates.

`examples/news-edition` as a separate Go module only imports the `extension` surface; a second theme and CI importguard/build proof were added with `EditorialTheme`. The internal SQL layer of the News module is not exposed to the Client Edition.
