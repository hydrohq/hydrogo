# ADR-0204: OpenTelemetry observability boundary

- **Status:** Accepted
- **Date:** 2026-09-21
- **Related:** ADR-0054, ADR-0065, ADR-0190

## Decision

HydroGo uses the OpenTelemetry API for request trace-context propagation. The
default global provider is intentionally no-op: Compact starts no exporter,
queue, or network client. Standard and Distributed deployments may configure a
provider through a future operator-owned adapter; Core does not embed a vendor
endpoint or credential.

HTTP spans contain only bounded method, path, and correlation ID attributes.
They never include query values, bodies, cookies, authorization headers,
actor/tenant identifiers, SQL, or media names. Exporter sampling, timeout,
retry, and redaction policy are operational configuration and require separate
production evidence before a tracing-service support claim.

## Validation

- The tracing middleware preserves request semantics under the no-op provider.
- Existing logging and Prometheus metrics remain independent fallbacks.
- Any exporter adapter requires bounded failure behavior and PII review.
