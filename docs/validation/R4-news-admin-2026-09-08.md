# R4 News admin flow validation

**Date:** 2026-09-08  
**Roadmap:** R4 News review/schedule/publish/retract  
**Status:** Succeeded; domain, generation guarded worker, durable scheduler bootstrap, JSON/public and protected admin HTML flows are ready.

The News admin screen executes draft edit, schedule, publish, and retract operations via the same `news.Service`. Every mutation form contains a CSRF token and an optimistic `version` field; editor/admin authorization is verified alongside the router's protected boundary. The schedule worker carries job generation in its payload, and the repository invalidates the old job via atomic generation check. `cmd/hydro serve` now queues due scheduled records into the durable `jobs` store using a five-second bounded polling and runs the lease worker; enqueue retries remain idempotent with the job ID.

Executed checks:

```text
go test ./modules/news/transport ./...
go vet ./...
```

The second external edition also applies the `EditorialDraft` rule without importing it into Core: title and desk are mandatory; editorial approval is required for scheduled news. `examples/news-edition/editorial_policy_test.go` validates this behavior. The remaining work for this gate is worker process segregation and capacity measurement in an actual deployment.
