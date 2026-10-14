# Prometheus-compatible metrics

When `server.Config.Metrics` is configured, HydroGo exposes the protected
`GET /metrics` route. The response uses the Prometheus text 0.0.4 format and
contains only fixed metric names and bounded labels (`1xx`–`5xx` response
class and latency bucket). If metrics are not configured, the route is not
registered.

The exporter is dependency-free and process-local. It intentionally does not
pretend to be an OpenTelemetry SDK; OTLP export remains an explicit deployment
integration. Protect `/metrics` at the application authorizer and at the
network boundary before scraping.
