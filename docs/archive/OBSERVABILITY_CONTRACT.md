# HydroGo observability contract

Metrics and logs use stable names and bounded dimensions. Request paths are
bounded and query strings, bodies, cookies, and response bodies are excluded.

| Signal | Contract | Cardinality rule |
|---|---|---|
| `hydro_http_requests_total` | request count by method/status class | method and class only |
| `hydro_http_duration_ms` | bounded request latency | no URL labels |
| `hydro_jobs_*` | queue depth and worker outcomes | no job ID labels |
| `hydro_readiness_*` | readiness state/failures | fixed labels only |
| `hydro_session_cleanup_*` | cleanup runs, rows, errors | fixed labels only |

New labels require an ADR or explicit review because unbounded tenant, user,
path, or error-string labels can exhaust metrics backends and expose PII.
Structured request logs must carry the request ID and redact secrets/PII.
