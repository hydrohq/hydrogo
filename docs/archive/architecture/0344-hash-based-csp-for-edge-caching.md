# ADR-0344: Hash-based CSP for edge caching

- **Status:** Proposed
- **Date:** 2026-10-03
- **Supersedes:** ADR-0329 (partially - the nonce generation requirement)
- **Superseded by:** None

## Context

The capacity benchmark results in `docs/validation/hydrogo-capacity-benchmark-2026-09-30.md` showed that deploying a reverse proxy/edge cache (like Caddy + Souin) in front of HydroGo could increase throughput dramatically (from ~1,400 req/s to ~6,184 req/s), while reducing HydroGo's process CPU utilization to near zero.

However, the experiment was rejected because HydroGo was using dynamically generated, per-request CSP (Content-Security-Policy) nonces for inline scripts and styles. When the HTML was cached at the edge, the same CSP nonce was replayed to all clients, defeating the per-response freshness guarantee and violating the security model defined in ADR-0329.

To safely enable aggressive edge caching and achieve 6000+ req/s without leaking security tokens, we must stop using per-request CSP nonces.

## Decision

We will replace dynamically generated CSP nonces (`'nonce-...'`) with hash-based CSP (`'sha256-...'`) for static inline scripts and styles.
1. The `internal/security/headers.go` middleware will no longer generate a random nonce per request.
2. The CSP header will explicitly list the SHA-256 hashes of the known inline scripts and styles used by the application (e.g., the HTMX initialization script, theme toggle scripts).
3. All UI components (`internal/ui`) must remove the `nonce` attribute from their `<script>` and `<style>` tags, as the browser will validate them against the hash provided in the CSP header.
4. If an inline script or style contains dynamic data that changes per request, it must be refactored into a separate `.js`/`.css` file or safely passed via `data-*` attributes to a static script, because dynamic inline scripts cannot be hashed ahead of time.

This explicitly supersedes the nonce-generation requirements in ADR-0329.

## Alternatives

- **Excluding nonce-bearing HTML from the edge cache:** Rejected because almost all public pages contain nonce-bearing scripts/styles, which would nullify the benefit of the edge cache.
- **Custom edge cache module:** Rejected because it would require building and maintaining a custom Caddy/Nginx module to rewrite nonces on the fly, adding unacceptable operational complexity.

## Consequences

### Benefits

- **Massive Performance Gain:** Public HTML can be safely cached by any generic RFC 7234 reverse proxy (Caddy, Nginx, Cloudflare), unlocking 6000+ req/s throughput.
- **Reduced CPU Load:** HydroGo process CPU utilization drops to near zero for cached responses.
- **Simplified Middleware:** We no longer generate random bytes or manage context values for nonces on every request.

### Costs and risks

- **Development Friction:** Any change to an inline script or style will require updating the corresponding SHA-256 hash in the CSP header.
- **No Dynamic Inline Scripts:** We lose the ability to easily inject request-specific dynamic data directly into inline scripts.

## Scale impact

- Compact: Lower memory and CPU footprint per request.
- Standard: Enables caching at the edge, massively increasing throughput.
- Distributed: Essential for CDN/Cloudflare caching at scale.

## Database impact

- SQLite: None.
- PostgreSQL: None.
- Future adapters: None.

## Cache impact

Allows unmodified external edge caches (CDN, Caddy, Nginx) to cache HTTP 200 HTML responses indefinitely according to the `Cache-Control` max-age, without compromising CSP security.

## Deployment and compatibility

Safe. External caches will begin caching the responses immediately.

## Validation

A load test with Caddy + Souin should reach 6000+ req/s while preserving CSP integrity and passing all security tests.

## References

- `docs/validation/hydrogo-capacity-benchmark-2026-09-30.md`
- ADR-0329
