# R3 — Dynamic redirect path segment validation

**Status:** Local contract succeeded.

Pages, News, Commerce, and Members admin mutation redirects now use
`url.PathEscape` for dynamic IDs. This keeps attacker-controlled separators
from changing the redirect route while preserving normal identifiers. Existing
transport suites continue to cover the redirect responses, and the member
transport test covers the related forged-Referer fallback.

Evidence: `go test ./modules/members/transport ./modules/pages/transport
./modules/news/transport ./modules/commerce/transport` passed on 2026-09-14.
Full repository pre-push verification is required before delivery.
