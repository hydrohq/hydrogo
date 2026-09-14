# ADR-0075: Keep audit and member PII behind explicit admin access

## Status

Accepted

## Decision

Audit/member detail routes require an active admin role before loading member,
session, or audit data. Rendered values remain HTML-escaped, logs continue to
omit email, IP, user-agent, request query/body, and credential material. Admin
detail rendering tolerates an unavailable optional session repository without
turning a read request into a server panic.

## Consequences

Operational logs stay low-PII while authorized Studio admins retain the minimum
identity and audit context needed for support. External privacy review and
retention policy decisions remain deployment/product gates.
