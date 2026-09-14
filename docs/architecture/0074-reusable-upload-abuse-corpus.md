# ADR-0074: Keep upload abuse checks in a reusable corpus

## Status

Accepted

## Decision

Maintain shared security corpus coverage for unsafe redirects/host confusion,
oversized bodies, traversal and disallowed uploads, image pixel bombs, corrupt
payloads, and malformed/oversized multipart requests. Tests assert fail-closed
HTTP status behavior and avoid exposing parser or storage internals.

## Consequences

New upload boundaries can reuse high-signal cases and regressions are visible
in one local verification path. A corpus does not replace external penetration
testing or provider-specific malware scanning.
