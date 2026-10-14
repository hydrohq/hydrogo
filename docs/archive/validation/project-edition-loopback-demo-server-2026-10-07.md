# Project Edition demo server bind safety

**Date:** 2026-10-07  
**Scope:** Prevent the permissive, test-only Project Edition admin composition
from being exposed on a public network interface.

The example's `Authorize` and CSRF callbacks are intentionally permissive to
exercise generated CRUD locally. `edition-server serve` now accepts only
`localhost`, IPv4 loopback, and IPv6 loopback addresses. Wildcard, LAN, public
IP, hostname, and malformed addresses are rejected before the database is
opened. The example README makes the demo-only policy boundary explicit.

`go test ./examples/project-edition/cmd/server -count=1` covers allowed and
rejected address classes and verifies `serve --address 0.0.0.0:8091` fails at
the address guard. The Chromium Edition fixture continues to connect over
`127.0.0.1`.

This is a safety guard for the example server; it is not an authentication
implementation. A deployable Edition still needs its own identity, role,
resource, CSRF, and tenant authorization.
