# Pointing a domain at Cloudflare and configuring SSL (AI guide)

Audience: an agent setting up DNS for a domain in front of a shared-hosting
or VPS origin, using Cloudflare as the proxy/CDN. Mail-specific DNS
records (SPF/DKIM/DMARC) are covered in `mail-smtp-setup.md` — this guide
is the general domain-pointing and SSL-mode setup.

## 1. Nameserver delegation

The domain's registrar must point its nameservers at the two Cloudflare
gives when the zone is added (e.g. `xxx.ns.cloudflare.com`,
`yyy.ns.cloudflare.com` — exact values come from the Cloudflare dashboard
for that zone, don't hardcode example ones). This is done at the
**registrar**, not in Cloudflare itself. Propagation is the one step here
genuinely outside your control — it can take anywhere from minutes to
~48 hours depending on the registrar and the previous DNS TTL.

### Diagnosing propagation issues

If the domain resolves via `curl` from one network but not another (e.g.
works on mobile data, not on home WiFi; or a WebFetch/agent-side lookup
gets `EAI_AGAIN`/`NXDOMAIN` while the user's browser works), this is
**expected** during propagation — different resolvers update at different
times. It's not a configuration bug to "fix"; the only real signals worth
checking are: the Cloudflare dashboard shows the zone status (not
"Pending Nameserver Update"), and a public DNS propagation checker
(`dnschecker.org` or similar) shows it resolving from multiple global
locations. Seeing the origin server's own access/error logs receiving
requests *from Cloudflare's IP ranges* (`104.x`, `172.x`, `162.158.x`,
etc. — Cloudflare's published ranges) is strong evidence the zone is
already active on Cloudflare's side even if your own resolver hasn't
caught up yet.

## 2. DNS records

```
A     @          <origin-ip>     Proxied (orange cloud)
A     www        <origin-ip>     Proxied (orange cloud)
```
Only proxy records that should actually go through Cloudflare's edge
(web traffic). Records serving a different purpose at the same apex
(mail, a bare API not meant to be cached/proxied) typically should be
**DNS only** (grey cloud) — see `mail-smtp-setup.md` for why this matters
specifically for DKIM.

## 3. SSL/TLS mode

Cloudflare → SSL/TLS → Overview. Use **Full (strict)** — this requires the
origin to present a valid certificate Cloudflare can verify (Let's
Encrypt via the hosting panel, or a Cloudflare Origin Certificate
installed on the origin). **Do not use "Flexible"** for an app that sets
cookies or checks `X-Forwarded-Proto`/HSTS — Flexible means the
Cloudflare-to-origin leg is plain HTTP, which can create redirect loops or
inconsistent secure-cookie behavior when the app's own HSTS/secure-cookie
settings assume the whole path is HTTPS.

If the origin doesn't have its own trusted cert yet: Cloudflare → SSL/TLS
→ Origin Server → Create Certificate generates a cert+key pair trusted by
Cloudflare (not publicly trusted elsewhere) — install it on the origin via
the hosting panel's SSL/TLS upload, which is sufficient for Full (strict)
since only Cloudflare needs to trust it.

Cloudflare → SSL/TLS → Edge Certificates → enable **"Always Use HTTPS"**.

## 4. Verifying end-to-end

```sh
curl -s -D - -o /dev/null "https://<domain>/" --max-time 15
```
Check for: `Server: cloudflare` header (confirms the proxy is in the
path), a valid `200`, and — if the app sets HSTS —
`strict-transport-security` present. `CF-RAY` header presence confirms
the request actually went through Cloudflare's edge, useful when
debugging whether a weird response came from Cloudflare or the origin
directly.

## What NOT to do

- Don't assume a `curl`/`WebFetch` DNS failure from your own environment
  means the domain setup is broken — cross-check with the origin's own
  logs and the Cloudflare dashboard before concluding anything is
  misconfigured.
- Don't leave SSL mode on Flexible "temporarily" — it's a common source of
  redirect loops that's easy to forget about once the site appears to work.
