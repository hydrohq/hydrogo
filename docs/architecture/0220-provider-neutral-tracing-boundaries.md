# ADR-0220: Provider-Neutral Tracing Boundaries

## Status

Accepted — 2026-09-22

## Decision

HTTP handling, database repositories/transactions, cache operations, worker jobs, webhooks, and backup tasks are annotated with provider-neutral span boundaries (`internal/telemetry` interfaces wrap database transactions). Exporters and sampling dependencies are not mandated in Core; the global OpenTelemetry provider is configured according to the deployment profile. Span attributes are restricted to normalized route, component, status, and bounded execution duration; raw queries, bodies, cookies, secrets, and high-cardinality IDs are strictly prohibited.

## Operations

The Compact profile runs with zero exporter overhead. Standard and Enterprise deployments configure the OTel SDK/exporter via standard deployment configuration specifying OTLP endpoints and sampling ratios. Collectors manage downstream retention and access policies.

`LoadTracingProfile` validates that endpoints are absolute URLs and sampling rates remain within the `[0.0, 1.0]` range.
