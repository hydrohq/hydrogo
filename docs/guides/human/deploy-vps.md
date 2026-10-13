# Deploying to a VPS (Human Guide)

This guide walks through deploying HydroGo as a persistent daemon service on a Linux VPS (DigitalOcean, AWS EC2, Hetzner, Linode, Ubuntu/Debian server, etc.) with SSH access. Unlike the shared hosting CGI model where a new process is spawned per request, the standalone daemon runs as a continuous system service behind a reverse proxy (systemd + Nginx).

## Pre-flight Checks: Inspect Existing VPS Services

Do not assume the VPS is empty. Verify existing services (Docker containers, existing web servers). In particular, check whether ports **80 and 443** are currently bound by another service.

## Deployment Overview

1. **Upload the Binary** — Cross-compile locally and transfer via `scp` or `rsync` (no Go toolchain required on the target server).
2. **Prepare `config.yaml`** — Set database dialect (SQLite works efficiently on a VPS without shared-hosting file contention), theme paths, and `public_origin`.
3. **Execute Migrations & Create Admin** — Run `hydro migrate` and `hydro create-admin` directly via terminal.
4. **Configure systemd Service** — Ensures automatic restart on server reboots or unexpected process termination.
5. **Configure Nginx Reverse Proxy** — Terminates traffic and forwards requests to HydroGo. **Critical requirement:** Ensure Nginx forwards the accurate `Host` header and `X-Forwarded-*` headers matching `public_origin`.
6. **Set up TLS/SSL** — Use Let's Encrypt (`certbot`) or Cloudflare SSL proxy.

## Cloud Provider Security Groups & Firewalls

A frequent troubleshooting scenario is where a service functions on `localhost` via curl, but external connections timeout. Cloud providers (AWS Security Groups, DigitalOcean Cloud Firewalls, Hetzner Firewalls, GCP Firewall Rules) enforce hardware-level network filters outside the OS `ufw`/`iptables`. Ensure incoming TCP ports 80 and 443 are allowed in your cloud provider's web console.

## Cleaning Up Test Deployments

If validating on a temporary or shared server, remove test systemd units, Nginx sites-enabled symlinks, and temporary databases once verification is complete.
