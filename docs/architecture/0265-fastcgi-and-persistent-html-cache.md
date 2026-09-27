# ADR-0265: FastCGI and persistent CGI HTML cache

- **Status:** Accepted
- **Date:** 2026-09-27
- **Supersedes:** [ADR-0264](0264-cgi-hosting-runtime.md)

## Context

CGI covers process-per-request hosting, while some shared hosts provide a
FastCGI manager. CGI also loses its in-memory page cache after every request.
The runtime needs to preserve the existing public HTML, security, and worker
contracts in both lifecycles.

## Decision

The executable supports CGI through `net/http/cgi` and FastCGI through
`net/http/fcgi`. CGI mode uses a one-request process and suppresses resident
workers. FastCGI is a long-running responder bound to loopback by default; it
uses the normal bounded worker lifecycle and drains active HTTP handlers on
shutdown. The hosting provider owns FastCGI process supervision and proxy
configuration.

In CGI mode, public page and news HTML responses are cached below
`DOCUMENT_ROOT/static_cache`, so Apache can serve them without starting CGI.
Only successful, anonymous, non-HTMX, query-free HTML responses no larger
than 512 KiB are eligible. The cache has a one-minute browser expiry and holds
at most 128 entries or 32 MiB. Only public page/news HTML routes are eligible;
responses that set cookies, request private/no-store caching, vary on other
headers, or contain a CSP nonce are skipped. The `.htaccess` denies direct
requests to `/static_cache/` and adds a fixed CSP to HTML responses. Page and
news mutations invalidate the cache. Out-of-band database writes require an
operator to clear this public HTML cache.

CGI lazy cron discovers due News schedules and processes at most four leased
jobs within a five-second request context. Other recurring work requires
FastCGI/`hydro serve` or an operator cron command; CGI does not create a hidden
daemon or unbounded process.

## Consequences

- CGI and FastCGI share the existing route authorization and server handler.
- No database migration is required; durable job leases remain the existing
  scheduling contract.
- FastCGI integration is a host configuration and is not assumed to be
  available in `.htaccess`.
- The public HTML cache is bounded by entry count, total size, and per-response
  size. In-app writes invalidate it; operators must clear it after out-of-band
  changes. If `DOCUMENT_ROOT` is unavailable, CGI still works without the
  static disk cache.
