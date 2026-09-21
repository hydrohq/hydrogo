# E0 OpenTelemetry HTTP redaction validation

- **Date:** 2026-09-21
- **Scope:** HTTP span attribute boundary.
- **Not evidence for:** an enabled production exporter or a tracing-service
  support/SLO claim.

`internal/observability` is tested with the OpenTelemetry SDK in-memory
exporter. A request containing a query token, Authorization header and session
cookie exports only the bounded method/path/request-ID attributes. The test
asserts that the token value and sensitive field names never enter the span.

The SDK is a test dependency for this assertion. HydroGo runtime still uses the
global no-op provider unless an operator-owned adapter is supplied in a future
decision; Compact therefore starts no telemetry queue or network client.

```text
go test ./internal/observability -count=1
```

Result: pass.
