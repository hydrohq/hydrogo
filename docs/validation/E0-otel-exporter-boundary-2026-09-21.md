# E0 OpenTelemetry Exporter Boundary Validation — 2026-09-21

- `internal/observability/exporter.go` provides an opt-in adapter with
  operator-owned sampling, a per-attempt timeout, and a maximum of five
  attempts.
- The adapter accepts only `SpanRecord`, the already-redacted provider-neutral
  shape; it does not start an exporter or add runtime infrastructure in Compact.
- `internal/observability/exporter_test.go` proves sampling excludes non-error
  spans and failed delivery is retried exactly to the configured bound.

The adapter is intentionally not wired to a vendor SDK. A deployment may wrap
its own exporter behind the interface after reviewing retention, PII and
network egress policy.
