# Transactional mail setup (SMTP relay + DNS) (AI guide)

Audience: an agent configuring outbound mail (password reset, admin
notifications, member verification) for a HydroGo deployment via an SMTP
relay provider (Resend used as the concrete example — the mechanism is
provider-agnostic SMTP, nothing Resend-specific in the app itself).

## HydroGo's mail config is plain SMTP, nothing provider-specific

`internal/config/config.go`'s `Mail` struct: `host`, `port`, `from`,
`username`, `password`, `starttls`, plus rate limits. `internal/mailer`
does not sign mail itself or integrate with any provider's API — it's a
standard SMTP client. `internal/mailer/dns.go`'s `DiagnoseDNS` is a
**read-only diagnostic** (checks SPF/DKIM/MX record presence for `hydro
doctor`-style output), not a signer — actual DKIM signing happens at the
SMTP relay, outside this app entirely.

## 1. Config

```yaml
mail:
  host: "smtp.resend.com"
  port: 587
  from: "<address>@<domain>"
  username: "resend"
  password: "<api-key>"
  starttls: true
```
`username` is literally the string `resend` for Resend specifically (not
an account email) — check the specific provider's SMTP docs for their
convention, don't assume it's always the provider name.

**Use port 587 + `starttls: true`, not port 465 + `starttls: false`.**
HydroGo's SMTP client implements the STARTTLS upgrade flow (plaintext
connect, then upgrade to TLS) — port 465's "implicit TLS" (TLS from the
first byte, no upgrade) is a different protocol-level handshake. Using 465
with `starttls: false` was tried and is not confirmed to work with this
client; 587+STARTTLS is the verified-working combination.

**YAML indentation**: a `mail:` block with a stray leading space on the
key (`" mail:"` instead of `"mail:"`) silently breaks the entire config
file and takes the whole site down (`hydro: decode config: yaml: line N:
did not find expected key`). If a config edit causes this, diff the
indentation of every top-level key against the rest of the file rather
than guessing which line is wrong.

## 2. DNS records the mail provider requires

Check the provider's own dashboard for the exact current records (don't
hardcode example values into production) — but the general shape, using
Cloudflare DNS as the example:

| Type | Name | Content | Proxy |
|---|---|---|---|
| TXT | `@` (or provider-specified) | SPF string the provider gives you | DNS only |
| TXT/CNAME | `<provider>._domainkey` or similar | DKIM key/CNAME the provider gives you | **DNS only, never proxied** |
| TXT | `_dmarc` | `v=DMARC1; p=none; rua=mailto:<address>` | DNS only |

**DKIM must never be Cloudflare-proxied (orange cloud).** If a DKIM record
is a CNAME and proxying is on, Cloudflare returns its own IP instead of
the real DKIM public key, and the signature can never verify — mail goes
out unsigned, lands in spam. Proxying is fine for `A`/`CNAME` records that
actually serve web traffic (`@`, `www`), never for mail-authentication
records.

## 3. SPF: exactly one record per domain

**A domain can have only one SPF record.** If the domain's mail is also
handled by something else (e.g. the hosting panel's own mail server), its
SPF (`v=spf1 +a +mx +ip4:<server-ip> ~all`) and the relay provider's SPF
requirement must be **merged into a single TXT record**, not added as two
separate ones:
```
v=spf1 +a +mx +ip4:<server-ip> include:<provider-required-include> ~all
```
Two separate `v=spf1` TXT records on the same name is an SPF spec
violation (`permerror`) and a strong spam signal — check for this
specifically if mail is landing in spam despite DKIM looking fine.

## 4. Diagnosing spam placement

In order of likelihood when mail lands in spam despite a config that looks
correct:
1. Duplicate/conflicting SPF records (see above) — check with a DNS lookup
   tool or the zone's record list directly, count how many `v=spf1` TXT
   records exist at the relevant name.
2. DKIM record proxied through Cloudflare (or similar) instead of
   DNS-only.
3. Leftover DKIM records from a previous/wrong provider account — if the
   provider's domain was set up more than once (e.g. wrong account first,
   correct account second), a stale TXT record from the first attempt can
   persist at the same name as the current correct one, with a different
   key, causing inconsistent verification. Check for duplicate records at
   the same DKIM selector name; if found, confirm with the operator which
   one is current (don't guess) and remove the stale one.
4. Domain not actually showing "Verified" in the provider's dashboard yet
   (DNS propagation or a missed record) — mail sent while unverified goes
   out unsigned.
5. `from` address domain doesn't match the verified sending domain.

## 5. Verifying after a fix

Send a real test email (trigger a password-reset or similar flow) and
check the delivered message's authentication results — in Gmail, "Show
original" / "View original" shows `SPF: PASS`, `DKIM: PASS`, `DMARC: PASS`
(or their failure reasons) directly. This is the only way to confirm a fix
actually worked — a "Verified" badge in the provider's dashboard confirms
DNS is readable, not that a specific message passed authentication.
