# R95 — Bounded outbox polling lifecycle

Date: 2026-09-19

## Evidence

`internal/webhooks.Worker.Run` wraps the lease-based `RunOnce` operation with a
bounded adaptive poll interval. Empty queues and transient database errors back
off from the configured base to a maximum interval; successful claims reset the
interval. Context cancellation stops the loop, and invalid/too-small/unbounded
intervals are rejected. The test uses a lock-like database error and proves the
10ms → 20ms → 40ms sequence without a tight spin.

Commands:

- `go test ./internal/webhooks -run TestWorkerRun -count=1`
- `go vet ./internal/webhooks`

Result: passed.

This is a worker-loop contract, not a production SQLite contention or capacity
certification. Those require representative data, disk and worker-count soak.
