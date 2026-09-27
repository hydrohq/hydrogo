# ADR-0264: CGI hosting runtime

- **Status:** Accepted
- **Date:** 2026-09-27

## Context

Shared hosting platforms can expose an executable as a CGI program without a
resident application process. CGI starts a process for each request, so the
normal HTTP listener and its long-lived workers cannot be reused as-is.

## Decision

When `GATEWAY_INTERFACE` is present, the HydroGo executable handles one request
through the standard library's `net/http/cgi` adapter. `hydro serve --cgi` is
also available for explicit invocation. CGI mode does not start the SQLite
external-write monitor, media derivative worker, session cleanup worker, news
polling worker, or webhook worker.

For editions with News enabled, each CGI request performs one due-schedule
discovery and processes at most four leased jobs, with a five-second context
budget. Durable job IDs and leases preserve idempotency across separate CGI
processes. Static files already present in the document root are left to
Apache, which may revalidate HTML files after one minute. The example rewrite
rules are in `deploy/hosting/.htaccess`.

This contract supports CGI only. FastCGI requires a persistent request loop and
is not provided by `net/http/cgi`; deployments needing it continue to use
`hydro serve` behind a reverse proxy or adapter.

## Consequences

- CGI uses the same server handler and route authorization as `hydro serve`.
- The hosting adapter adds no resident service or schema migration.
- CGI processes perform bounded scheduling work during requests; other
  background tasks require the normal long-running server or an operator cron.
- Existing files bypass CGI and are served directly by the web server.
