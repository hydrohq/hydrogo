# ADR-0063: Correlate request, actor, and session fields in logs

## Status

Accepted

## Decision

The HTTP request log event includes the existing bounded request ID and, when
available, opaque actor and session identifiers from auth context. It continues
to omit query strings, bodies, cookies, emails, token hashes, and IP addresses.

## Consequences

Operators can follow an authenticated request without receiving PII or secret
material in the structured event. Background job correlation remains a
separate contract because jobs do not yet carry an established context ID.
