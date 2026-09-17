# ADR-0142: Distributed profile is a certified topology, not a config label

- **Status:** Accepted (scope boundary); certification remains open

## Decision

- Distributed support requires PostgreSQL, shared Redis/Valkey-compatible L2, object storage, CDN, multiple web/worker instances, and measured backpressure.
- Local media, process-local queues, and process-local cache are not silently promoted to Distributed semantics.
- A release may expose the profile only after fan-out/reconnect, lost-invalidation, cache stampede, worker lease, backup/restore, S3/CDN, and capacity/soak drills pass.
- Compact and Standard remain independently usable without Distributed services.

## Release gate

The gate is evidence-based: topology manifest, p95/p99 and RSS/FD measurements, failure injection results, rollback/blue-green proof, and operator runbook with alert thresholds.
