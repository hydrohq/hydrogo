# ADR-0065: Keep metrics export explicit and bounded

## Status

Accepted

## Decision

HydroGo emits a dependency-free Prometheus-compatible text endpoint only when
the application supplies a metrics instance. The endpoint is protected and
uses fixed, bounded labels. OpenTelemetry export is not silently enabled.

## Consequences

Compact profiles incur no exporter route or formatting work when metrics are
disabled. Deployments choose their scrape authorization and any OTLP bridge
explicitly.
