# ADR-0063: Correlate request, actor, and session fields in logs

## Status

Accepted

## Decision

The HTTP request log event includes the existing bounded request ID and, when
available, opaque actor, session, and job/event identifiers from context. Job
workers propagate only the durable ID. It continues to omit query strings,
bodies, cookies, emails, token hashes, payloads, and IP addresses.

## Consequences

Operators can follow an authenticated request or background delivery without
receiving PII or secret material in the structured event.
