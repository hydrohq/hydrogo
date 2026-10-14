# Mail (SMTP) Setup Guide (Human Guide)

To reliably deliver transactional system emails (password resets, invitation links, form lead notifications), HydroGo connects to any standard SMTP relay service (such as Resend, Postmark, SendGrid, Amazon SES, or Mailgun). HydroGo acts as a standard SMTP client; email authentication (DKIM signing) is executed on the relay provider's mail servers based on your domain's DNS records.

## 1. Configure `config.yaml`

Using credentials from your email delivery provider (e.g., Resend):

```yaml
mail:
  host: "smtp.resend.com"
  port: 587
  from: "info@your-domain.com"
  username: "resend"
  password: "re_xxxxxxxxxxxxxxxx"
  starttls: true
```

*Note:* Port 587 with `starttls: true` is the standard verified configuration. Ensure YAML indentation uses clean spacing.

## 2. Configure Provider DNS Records

When onboarding your sending domain in your email provider's console, you will receive DNS records (SPF TXT, DKIM CNAME/TXT, and optional DMARC).

> [!IMPORTANT]
> **Mail DNS records (MX, SPF, DKIM) must NEVER be proxied through Cloudflare (no orange cloud).** Set them to **DNS-Only (grey cloud)**. Proxying DKIM or MX records breaks cryptographic domain verification, causing outgoing mail to fail spam filters.

## 3. The Single SPF Record Rule

RFC 7208 strictly specifies that a domain may publish **at most one** SPF TXT record. If you have existing mail servers (e.g., hosting server webmail), **do not add a second SPF TXT record**. Instead, merge them into a single string:

```text
v=spf1 +a +mx +ip4:<server-ip> include:resend.com ~all
```

Publishing multiple SPF records causes recipient MTAs (like Gmail and Outlook) to treat SPF verification as `permerror`, classifying your emails as spam or outright rejecting them.

## 4. Deliverability Troubleshooting Checklist

If emails land in spam folders or fail delivery, verify in order:
1. **Conflicting SPF records:** Confirm only a single `v=spf1` TXT record exists for the apex domain.
2. **Proxied DKIM:** Ensure DKIM CNAME/TXT records are set to "DNS Only" in Cloudflare.
3. **Stale DNS keys:** Check that no orphaned DKIM keys from previous test accounts exist.
4. **Domain verification status:** Confirm your domain is marked "Verified" in your email provider's dashboard.
5. **Sender address alignment:** Verify the `from` email domain exactly matches the authenticated domain.

## 5. End-to-End Verification

Trigger a live transactional email (such as requesting a password reset). In Gmail, click the menu (three dots) → **"Show original"**. Confirm that the headers display:
- `SPF: PASS`
- `DKIM: PASS`
- `DMARC: PASS`
