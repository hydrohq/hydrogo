# Deploy to a VPS via SSH (AI guide)

Audience: an agent with SSH/`scp` access to a Linux VPS. Produces a
long-running `hydro serve` process behind an nginx reverse proxy, managed
by systemd. Verified end-to-end on a real Ubuntu 26.04 VPS (2 vCPU, 908MB
RAM, 15GB disk) — every command below was actually run, not just planned.

Unlike shared/CGI hosting (`deploy-ftp-shared-hosting.md`), a VPS gives you
a persistent process: no per-request process-spawn cost, a real connection
pool, resident background workers all active, in-memory caching intact.
This is the deployment model HydroGo's architecture is actually built for.

## 0. Before touching anything: survey what's already there

```sh
ssh <user>@<host> "ls -la ~/ && sudo docker ps 2>/dev/null && sudo ss -tlnp | grep -E ':80 |:443 '"
```
A VPS handed to you for a deploy task is not necessarily empty. Check for:
- An existing checkout at a path you were about to use — don't `mkdir -p`
  into it assuming it's new; inspect it first (`git remote -v`, `git log -1`,
  or just `ls` if it's not a git repo) and use a different directory if it
  looks unrelated to your task.
- **Anything already bound to port 80/443** (`ss -tlnp`) — Docker containers
  commonly do this via `docker-proxy`. If something's already there, do not
  stop it; pick a different port for your own verification and document
  that the operator needs to choose a real port/vhost for production use.
- Low free disk (`df -h /`) — a small VPS can be at 90%+ already. `apt-get
  clean` before installing anything if it's tight; nginx itself is small
  (tens of MB with deps) but don't assume headroom.

## 1. Get the binary onto the VPS

No need to install a Go toolchain on the VPS — cross-compile locally and
`scp` the binary:
```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o dist/hydro ./cmd/hydro
scp dist/hydro <user>@<host>:<deploy-dir>/hydro
scp -r themes/<theme> <user>@<host>:<deploy-dir>/themes/<theme>
ssh <user>@<host> "chmod 755 <deploy-dir>/hydro"
```
**Theme directory structure matters**: `config.yaml`'s `theme_dir` must
point at the *parent* of a folder named exactly the `theme` value — i.e.
`theme_dir: "<deploy-dir>/themes"` + `theme: "corporate"` expects
`<deploy-dir>/themes/corporate/theme.json` to exist, not
`<deploy-dir>/themes-corporate/`. Get this wrong and the app still starts
but serves broken/missing theme assets.

## 2. `config.yaml`

```yaml
profile: compact
modules: [core, auth, members, pages, media, site, news]
http:
  address: "127.0.0.1:8080"
  public_origin: "http://<vps-ip-or-domain>"   # https:// once TLS is terminated at the proxy
database:
  dialect: "sqlite"       # or mysql/postgres if the VPS has one; fine either way on a VPS (real connection pool, no CGI locking concern)
  dsn: "<deploy-dir>/data/hydro.db"
media:
  dir: "<deploy-dir>/media"
theme_dir: "<deploy-dir>/themes"
theme: "<theme>"
security:
  secure_cookies: false   # true once HTTPS is actually terminated — see step 6
  enable_hsts: false
  headers_preset: strict
  enable_anomaly_filter: true
```

## 3. Validate, migrate, create the admin account

```sh
ssh <user>@<host> "cd <deploy-dir> && ./hydro validate --config config.yaml"
ssh <user>@<host> "cd <deploy-dir> && ./hydro migrate --config config.yaml"
ssh <user>@<host> "cd <deploy-dir> && printf '%s\n' '<password>' | ./hydro create-admin --email <email> --password-stdin --config config.yaml"
```
Same commands as shared hosting, but you have a real shell now — no
token-gated CGI workaround needed.

## 4. systemd unit

```ini
[Unit]
Description=HydroGo
After=network.target

[Service]
Type=simple
User=<user>
WorkingDirectory=<deploy-dir>
ExecStart=<deploy-dir>/hydro serve --config <deploy-dir>/config.yaml
Restart=on-failure
RestartSec=2
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=<deploy-dir>
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```
`ProtectSystem=strict` + explicit `ReadWritePaths` is worth keeping — the
process only needs write access to its own data/media directories, nothing
else on the filesystem.

```sh
ssh <user>@<host> "sudo tee /etc/systemd/system/hydrogo.service >/dev/null < unit-file && sudo systemctl daemon-reload && sudo systemctl enable --now hydrogo.service"
```
Verify: `sudo systemctl status hydrogo.service` should show `active
(running)`. Memory footprint observed on the real test: ~80MB RSS at idle.

## 5. nginx reverse proxy

```nginx
server {
    listen 80;
    server_name <domain-or-ip>;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```
**`proxy_set_header Host $host` is not optional.** `hydro serve` validates
the incoming `Host` header against `http.public_origin` and returns `421
Misdirected Request` on a mismatch — confirmed by testing directly against
`127.0.0.1:8080` with curl's default Host header before nginx was in the
loop. nginx forwarding the real client Host transparently is what makes
this work without any app-side config change.

```sh
sudo ln -sf /etc/nginx/sites-available/<name> /etc/nginx/sites-enabled/<name>
sudo nginx -t && sudo systemctl reload nginx
```

## 6. TLS

Terminate TLS at nginx (`certbot --nginx` for Let's Encrypt is the standard
path) or at a CDN/proxy in front of the VPS (Cloudflare, same pattern as
the shared-hosting guide). Once HTTPS is real, flip `secure_cookies: true`
and `enable_hsts: true` in `config.yaml` and restart the service.

## 7. Verify

From the VPS itself first (rules out app/proxy problems before blaming the
network):
```sh
ssh <user>@<host> "curl -s -o /dev/null -w '%{http_code}\n' -H 'Host: <domain-or-ip>' http://127.0.0.1:<nginx-port>/"
```
Then from outside:
```sh
curl -s -o /dev/null -w '%{http_code}\n' "http://<domain-or-ip>:<port>/"
```
**If the VPS-local check succeeds (200) but the external check times out**,
that is almost always the cloud provider's firewall (AWS Security Group,
GCP firewall rule, etc.) — a layer SSH and `ufw`/`iptables` cannot see or
fix. `ufw status` showing `inactive` does NOT mean the port is reachable
from the internet; the provider's own console/API has a separate inbound
rule set that must explicitly allow the port. This is provider-specific
and outside what you can verify or fix over SSH — tell the operator to
check their security group/firewall rules for the port in question.

## Cleanup discipline

If this was a verification/test run rather than the real production
deploy, remove everything you added before finishing:
```sh
sudo systemctl stop <service> && sudo systemctl disable <service> && sudo rm /etc/systemd/system/<service>.service && sudo systemctl daemon-reload
sudo rm /etc/nginx/sites-enabled/<name> /etc/nginx/sites-available/<name>
rm -rf <deploy-dir>
```
Confirm any pre-existing services on the box (Docker containers, other
systemd units) are still running afterward — don't just assume you didn't
touch them, check (`docker ps`, `systemctl status <their-unit>`).
