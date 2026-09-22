# Observability profile and retention

HydroGo emits one bounded JSON request event per request. Allowed fields are
`request_id`, optional `trace_id`, method, normalized `route`, bounded
`module`, status, duration, request/response bytes, and validated tenant/user
identifiers. Query strings, bodies, cookies, email addresses, tokens and
arbitrary exception text are not emitted.

## Profiles

- **Compact:** in-memory endpoint metrics, no OTLP exporter, bounded ring
  samples; restart resets process telemetry.
- **Standard:** the deployment may install an OpenTelemetry provider and set
  `HYDRO_OTEL_ENABLED=true`, `OTEL_EXPORTER_OTLP_ENDPOINT` and a sampling
  argument between `0` and `1`.
- **Enterprise:** use a managed collector with tenant-aware access control,
  encryption at rest and a documented export policy. HydroGo does not hold KMS
  keys.

The log sink/operator must configure retention and deletion according to the
tenant's legal policy. A practical default is 30 days for request logs and 90
days for aggregate content-view rows; this is a deployment decision, not a
hard-coded data lifecycle promise. Content analytics requires explicit,
purpose-versioned consent and stores no visitor identity.
