# ADR-0310: Mail sender DNS diagnostics

## Status

Accepted

## Decision

The authenticated Mailer settings surface provides a read-only check of SPF,
DMARC, DKIM, and MX records for the configured sender domain. DKIM uses the
configured `mail.dkim_selector` value and defaults to `default`. DNS work uses
the Go resolver under a two-second request deadline; it never delays message
delivery or creates background jobs. Results contain only record presence and
safe corrective guidance, not DNS response payloads or SMTP credentials.

SQLite/PostgreSQL behavior is unaffected because this capability is read-only
and has no persisted state. It adds a protected route only when the existing
Mailer and Auth modules are active. Rollback removes the route and page; no data
recovery is needed.

## Validation

Resolver tests cover sender-domain extraction, selector use, missing-record
guidance, and invalid sender settings. The rendered typed Design System view is
tested for HTML escaping.
