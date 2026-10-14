# ADR-0218: Structured Request Telemetry Dimensions

## Status

Accepted — 2026-09-22

## Decision

HTTP logs and tracing spans share bounded, canonical dimensions: `request_id`, `trace_id` (when active), normalized `route`, HTTP method, status code, duration in milliseconds, request/response byte counts, and authenticated tenant/user IDs. Query strings, request bodies, cookies, email addresses, auth tokens, and all other PII must never be emitted into telemetry under any circumstances. Tenant and user fields are populated strictly from authenticated request context.

## Consequences

- Endpoints correlate cleanly with logging streams without incurring high-cardinality explosions.
- When no distributed tracer exporter is configured, `trace_id` remains empty; Compact profile deployments incur zero telemetry network overhead.
- Log sinks independently enforce log retention and access control policies.

## Verification

- `internal/observability/logging_test.go` verifies normalized routes, tenant IDs, trace IDs, and byte counters; tests confirm PII scrubbing remains active.
