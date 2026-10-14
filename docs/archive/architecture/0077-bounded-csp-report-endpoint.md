# ADR-0077: Accept bounded CSP reports without persistence or CSRF state

## Status

Accepted

## Decision

Expose a public `POST /csp-report` telemetry endpoint that accepts both legacy
and modern CSP report shapes, caps the body at 16 KiB, validates line/column and
string boundaries, ignores unknown browser fields, and returns no content. The
endpoint bypasses CSRF because it does not mutate application state; reports
are neither persisted nor logged.

## Consequences

Generated pages can report policy regressions without leaking report contents
through logs or creating an application data store. Aggregation and alerting
remain operator-owned integrations.
