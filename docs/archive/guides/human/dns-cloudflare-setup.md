# Domain DNS & Cloudflare SSL Setup (Human Guide)

## 1. Update Nameservers

At your domain registrar (Namecheap, GoDaddy, Google Domains, Cloudflare Registrar, etc.), update the custom nameservers to the two nameserver addresses provided by Cloudflare during domain onboarding. Nameserver propagation across global ISPs typically takes from a few minutes up to 24–48 hours.

You can verify global propagation status using tools like `dnschecker.org`.

## 2. Configure DNS Records

In Cloudflare DNS Management, add your host routing records:

```text
Type   Name   Content       Proxy Status
A      @      <server-ip>   Proxied (Orange cloud)
CNAME  www    @             Proxied (Orange cloud)
```

*Note:* Mail records (MX, SPF TXT, DKIM CNAME/TXT) must **never be proxied** (DNS-only / Grey cloud). See [`docs/guides/human/mail-smtp-setup.md`](mail-smtp-setup.md) for mail DNS configuration.

## 3. SSL/TLS Encryption Mode

Navigate to Cloudflare → SSL/TLS → Overview:
- Select **"Full (strict)"** mode. Avoid "Flexible" mode, as Flexible terminates TLS at Cloudflare and communicates with your origin unencrypted, causing redirect loops with HydroGo's secure cookie and HTTPS redirection middleware.

If your origin server does not yet have a trusted TLS certificate:
- Provision a free **Let's Encrypt** certificate via your host or `certbot`, OR
- Navigate to Cloudflare → SSL/TLS → Origin Server → **Create Certificate**, and install the 15-year origin certificate onto your host.

Under Cloudflare → SSL/TLS → Edge Certificates:
- Enable **"Always Use HTTPS"**.

## 4. Verification

Navigate to `https://your-domain.com/` in your browser. Verify the TLS padlock icon appears without security warnings. If an error appears, verify that your Cloudflare zone status displays "Active" rather than "Pending Nameserver Update".
