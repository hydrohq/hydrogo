# ADR-0215: Bounded Endpoint Observability Contract

- **Status:** Accepted
- **Date:** 2026-09-22
- **Decision:** HydroGo exposes bounded, process-local endpoint reports using
  normalized API route labels and a fixed latency sample window. Reports include
  request/view/error counts, status classes, P50/P95/P99 and average latency,
  request/response bytes, timeout and rate-limit counts. Raw IDs and query
  strings are never exported as labels.

## Rationale

The existing process metrics are intentionally Compact-friendly and reset on
restart. This change improves endpoint diagnosis without introducing an
unbounded in-memory time series or making a remote metrics system mandatory.
The protected admin report and JSON/CSV export can expose the bounded snapshot;
Prometheus retains aggregate process counters for scraping.

## Cardinality and privacy rules

- API resource IDs after `/api/<resource>` or `/api/v1/<resource>` become
  `{id}` in endpoint reports.
- Query strings, bodies, cookies, authorization headers and visitor identities
  are excluded.
- Each route retains at most 128 latency samples and the registry retains at
  most 64 routes.
- Content views remain operational telemetry; unique-visitor analytics needs a
  separately consented, durable analytics module.
