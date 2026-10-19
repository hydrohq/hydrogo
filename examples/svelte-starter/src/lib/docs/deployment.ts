import { type DocSection, page, p, h2, h3, code, tabs, note, tip, warn, table, ul, ol } from './types';

export const deployment: DocSection = {
  id: 'deployment',
  title: 'Deployment',
  pages: [
    page(
      'vps',
      'VPS with systemd and a proxy',
      "Run HydroGo on any Linux server as a systemd service behind Caddy or nginx for TLS.",
      p("HydroGo is one static binary plus a config file and a data directory. On a VPS the usual shape is: `hydro serve` bound to localhost, managed by systemd, with Caddy or nginx terminating TLS in front."),
      h2('1. Install and create the project'),
      code('bash', `curl -fsSL https://github.com/ozyildirims/hydrogo/releases/latest/download/install.sh | sh
sudo useradd --system --home /srv/mysite --shell /usr/sbin/nologin hydro
hydro new mysite --preset corporate --admin-email you@example.com
sudo mv mysite /srv/mysite && sudo chown -R hydro:hydro /srv/mysite`),
      note("`hydro new` refuses to write into a folder that already exists, so scaffold first and move the result into place (or build your project on your laptop and copy it)."),
      h2('2. Production configuration'),
      p("Set these in `config.yaml` or as `HYDRO_*` variables (see [Environment variables](doc:configuration/environment-variables)):"),
      code('yaml', `profile: standard
http:
  address: "127.0.0.1:8080"
  public_origin: "https://cms.example.com"
  behind_proxy: true
security:
  secure_cookies: true
  enable_hsts: true`, 'config.yaml'),
      note("Enrol administrators in MFA before you enable `security.mfa_required_roles`."),
      h2('3. systemd unit'),
      code('ini', `# /etc/systemd/system/hydrogo.service
[Unit]
Description=HydroGo CMS
After=network-online.target
Wants=network-online.target

[Service]
User=hydro
WorkingDirectory=/srv/mysite
EnvironmentFile=-/etc/hydrogo/env
ExecStartPre=/usr/local/bin/hydro migrate --config config.yaml
ExecStart=/usr/local/bin/hydro serve --config config.yaml
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
ProtectSystem=full
ReadWritePaths=/srv/mysite

[Install]
WantedBy=multi-user.target`, 'hydrogo.service'),
      code('bash', `sudo systemctl daemon-reload
sudo systemctl enable --now hydrogo
sudo journalctl -u hydrogo -f`),
      tip("Put secrets (DSN, keys) in `/etc/hydrogo/env` with mode `0600`, owned by root, instead of in the unit file or `config.yaml`."),
      h2('4. TLS proxy'),
      tabs(
        {
          label: 'Caddy',
          lang: 'caddyfile',
          code: `cms.example.com {
    encode zstd gzip
    header {
        Strict-Transport-Security "max-age=63072000; includeSubDomains"
        X-Content-Type-Options "nosniff"
        -Server
    }
    reverse_proxy 127.0.0.1:8080 {
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-Proto https
    }
}`,
          title: 'Caddyfile',
        },
        {
          label: 'nginx',
          lang: 'nginx',
          code: `server {
    listen 443 ssl http2;
    server_name cms.example.com;

    ssl_certificate     /etc/letsencrypt/live/cms.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cms.example.com/privkey.pem;

    client_max_body_size 64m;   # media uploads

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}

server {
    listen 80;
    server_name cms.example.com;
    return 301 https://$host$request_uri;
}`,
          title: 'nginx.conf',
        },
      ),
      p("The repository contains complete, hardened examples in `deploy/caddy/Caddyfile` and `deploy/nginx/nginx.conf`."),
      h2('Built-in TLS'),
      p("Without a proxy, HydroGo can obtain certificates itself: set `http.auto_tls: true` and `http.domain: cms.example.com` (ports 80 and 443 must be reachable). It cannot be combined with `http.behind_proxy`."),
      h2('Verify'),
      code('bash', `curl -fsS https://cms.example.com/readyz
curl -fsS https://cms.example.com/api/v1/news?locale=en\\&limit=1
hydro doctor --config /srv/mysite/config.yaml`),
    ),

    page(
      'docker',
      'Docker',
      "Run the published image, or the production Compose stack with Caddy, PostgreSQL and Redis.",
      h2('Run the image'),
      code('bash', "docker run -p 8080:8080 -v hydro-data:/app/data ghcr.io/ozyildirims/hydrogo:latest"),
      p("Images are published to `ghcr.io/ozyildirims/hydrogo` for linux/amd64 and arm64. Tags follow releases (`v0.2.0`) and `latest`; pre-release tags (with a hyphen, like `v0.2.0-rc.1`) never move `latest`. Pin a version in production."),
      p("Mount a project folder at `/app` (it holds `config.yaml`, `content-types/` and `data/`), or just a config file at `/app/config.yaml`, and keep the data on a volume:"),
      code('bash', `docker run -d --name hydrogo -p 8080:8080 \\
  -v "$(pwd)":/app \\
  -e HYDRO_PUBLIC_ORIGIN=https://cms.example.com \\
  ghcr.io/ozyildirims/hydrogo:v0.2.0`),
      note("The Dockerfile in `deploy/` builds a non-root image whose working directory is `/var/lib/hydrogo`. Whichever image you use, the rule is the same: mount a persistent volume where your `database.dsn` and `media.dir` point."),
      h2('Production Compose stack'),
      p("`deploy/docker-compose.prod.yml` is a complete example: Caddy for TLS, HydroGo, PostgreSQL and Redis on a private network, with credentials supplied as Docker secrets. The entrypoint reads secret files and exports the matching `HYDRO_*` variables (database DSN, Redis URL, MFA key, S3 and SMTP credentials)."),
      code('bash', `cd deploy
export HYDRO_PUBLIC_HOST=cms.example.com
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml ps`),
      ul(
        "The app container runs as a non-root user (`10001`).",
        "Its health check calls `GET /readyz`; Caddy waits for it before accepting traffic.",
        "Run `hydro migrate` as an explicit operator step before starting a new version; runtime start-up does not apply schema changes on its own in this stack.",
        "Roll back by redeploying the previous image tag, never by deleting tables.",
      ),
      h2('Docker health check'),
      p("The binary has a built-in probe usable in `HEALTHCHECK` lines without curl: `hydro healthcheck` requests `http://127.0.0.1:8080/healthz` (override with `HYDRO_HEALTHCHECK_URL`)."),
      code('text', "HEALTHCHECK --interval=15s --timeout=4s CMD [\"hydro\", \"healthcheck\"]", 'Dockerfile'),
    ),

    page(
      'shared-hosting',
      'Shared hosting (CGI and FastCGI)',
      "Deploy on Apache-style shared hosting where you cannot run a long-lived process.",
      p("Many shared plans cannot keep a daemon running. HydroGo has two compatibility modes for them, at the cost of some performance. Use a VPS if you can; use these when you cannot."),
      h2('Build for the host'),
      p("Cross-compile a static, CGO-free `linux/amd64` binary (shared hosts have no C toolchain):"),
      code('bash', "make build-hosting        # Linux and macOS\n# Windows: scripts/build-hosting.ps1"),
      h2('CGI mode'),
      p("CGI starts a fresh process for each request, so there are no resident workers. The wrapper script is what Apache executes; keep the binary and `config.yaml` **outside the document root** (for example a sibling `private/` folder)."),
      code('bash', `#!/bin/sh
exec /path/to/private/hydro serve --cgi --config /path/to/private/config.yaml`, 'hydro.cgi'),
      ul(
        "Upload `hydro.cgi` through SFTP or the panel's upload function, not by pasting into an in-panel text editor. Files created in a panel can get different ownership and fail `suexec` checks with `Permission denied`.",
        "Add `DirectoryIndex hydro.cgi` to `.htaccess`. Without it, a request for `/` may return 403 once any placeholder `index.html` is gone.",
        "Apache serves existing files directly and sends everything else to HydroGo.",
        "Scheduled news jobs run in bounded batches during public requests.",
      ),
      warn("Under CGI every request opens its own database connection. SQLite's single-writer lock is risky under concurrency, so prefer MySQL or PostgreSQL when your plan offers it, and keep SQLite for low-traffic, single-editor sites."),
      h3('Running migrate without SSH'),
      p("If you have no shell, a one-shot CGI script can run `hydro migrate`. Protect it with a long random token, return 404 on mismatch, and delete it right after use."),
      code('bash', `#!/bin/sh
TOKEN="<random-hex>"
case "$QUERY_STRING" in
  token=$TOKEN)
    echo "Content-Type: text/plain"; echo ""
    /path/to/private/hydro migrate --config /path/to/private/config.yaml 2>&1
    ;;
  *)
    echo "Status: 404 Not Found"; echo "Content-Type: text/plain"; echo ""; echo "not found"
    ;;
esac`, 'migrate.cgi (delete after use)'),
      h2('FastCGI mode'),
      p("FastCGI is a long-running responder: start it under the provider's process manager and point Apache's FastCGI integration at the loopback address. The port is not bound to public interfaces."),
      code('bash', "hydro serve --fastcgi --fastcgi-address 127.0.0.1:9001"),
      p("HydroGo keeps its bounded background workers in this mode and drains requests on shutdown. The full guide, including `.htaccess` examples and the static page cache that lets Apache answer eligible public pages without starting CGI, is in `deploy/hosting/README.md`."),
    ),

    page(
      'databases-and-storage',
      'Databases, Redis and media storage',
      "Choose SQLite, PostgreSQL or MySQL, add Redis for several instances, and move media to S3.",
      h2('Databases'),
      table(
        ['Dialect', 'Use when', 'DSN example'],
        ['`sqlite` (default)', 'One instance, small to medium traffic, simple backups.', '`data/hydro.db`'],
        ['`postgres`', 'More traffic or several instances.', '`postgres://hydro_app:secret@db.internal:5432/hydro?sslmode=verify-full`'],
        ['`mysql` (and MariaDB)', 'You already run MySQL, or your host provides it.', '`hydro:secret@tcp(db.internal:3306)/hydro`'],
      ),
      code('bash', `HYDRO_DATABASE_DIALECT=postgres
HYDRO_DATABASE_DSN=postgres://hydro_app:REPLACE_ME@db.internal:5432/hydro?sslmode=verify-full
hydro migrate --config config.yaml`),
      p("Every migration ships SQL for all three dialects, so you can change database later by exporting and importing content (see `hydro portable export` and `hydro portable import`) into a freshly migrated database."),
      h3('SQLite notes'),
      ul(
        "Keep the database on local disk, not a network share.",
        "Only HydroGo should write to the file; set `database.sqlite_external_write_watch: true` if other tools might.",
        "Back it up with `hydro backup sqlite`, not by copying a live file.",
      ),
      h2('Redis or Valkey'),
      p("Optional for one instance, recommended for several. With `redis.url` set, rate limits and caches are shared, so counters are correct no matter which instance answers."),
      code('bash', "HYDRO_REDIS_URL=rediss://:REPLACE_ME@redis.internal:6379/0\nHYDRO_REDIS_KEY_PREFIX=hydro:"),
      h2('Media on S3'),
      p("By default media lives in `media.dir` on local disk. For multiple instances, containers without volumes, or large libraries, use an S3-compatible bucket (AWS S3, MinIO, Cloudflare R2, and similar) and, optionally, a CDN."),
      code('bash', `HYDRO_MEDIA_STORAGE_DRIVER=s3
HYDRO_MEDIA_S3_ENDPOINT=s3.eu-central-1.amazonaws.com
HYDRO_MEDIA_S3_BUCKET=example-media
HYDRO_MEDIA_S3_REGION=eu-central-1
HYDRO_MEDIA_S3_ACCESS_KEY=REPLACE_ME
HYDRO_MEDIA_S3_SECRET_KEY=REPLACE_ME
HYDRO_MEDIA_CDN_BASE_URL=https://cdn.example.com
# self-hosted stores often need path-style addressing
HYDRO_MEDIA_S3_FORCE_PATH_STYLE=true`),
      note("Moving existing local media to S3 means uploading the files to the bucket under the same keys. Test on a copy first."),
      h2('Sizing guidance'),
      table(
        ['Setup', 'Profile', 'Database', 'Extras'],
        ['Personal site, one editor', '`compact`', 'SQLite', 'none'],
        ['Company site or blog', '`standard`', 'SQLite or PostgreSQL', 'CDN for media'],
        ['High traffic or redundancy', '`distributed`', 'PostgreSQL or MySQL', 'Redis, S3, several instances'],
      ),
    ),

    page(
      'backups',
      'Backups and restore',
      "Back up the database and media, verify them, and practise a restore before you need one.",
      p("A backup you have never restored is a hope, not a backup. Schedule backups, copy them off the machine and rehearse the restore on a scratch copy."),
      h2('Back up'),
      code('bash', `# SQLite database plus media directory
hydro backup sqlite --destination backups/ --media-dir data/media --config config.yaml

# PostgreSQL
hydro backup postgres --destination backups/ --config config.yaml`),
      p("Run it from cron or a systemd timer:"),
      code('ini', `# /etc/systemd/system/hydrogo-backup.timer
[Timer]
OnCalendar=*-*-* 03:15:00
Persistent=true
[Install]
WantedBy=timers.target`, 'hydrogo-backup.timer'),
      code('bash', "# keep the 14 newest, preview first\nhydro backup prune --root backups/ --keep 14 --dry-run\nhydro backup prune --root backups/ --keep 14 --confirm"),
      p("For MySQL or MariaDB use your own `mysqldump`. For PostgreSQL you can also use `pg_dump` directly."),
      h2('Scheduled cloud backups'),
      p("With S3 configured you can enable scheduled backups inside HydroGo (`backup.enabled`, `prefix`, `retention_days`, `hour_utc`). The admin's **Telemetry > Backups** tab lists and verifies signed backups; sign them with `HYDRO_BACKUP_SIGNING_SEED`."),
      h2('Verify and restore'),
      code('bash', `hydro restore verify --snapshot backups/hydro-2026-04-01.snapshot

# unpack an archive into a NEW directory
hydro restore unpack --archive backups/site.tar.gz --destination restore/ --public-key HEX`),
      warn("Restores are deliberately guarded. `hydro restore stage` and `hydro restore postgres` require explicit flags confirming that sessions were revoked, egress is quarantined and old writes are blocked, so a restore cannot silently replay old sessions or double-write."),
      h2('Portable export'),
      p("`hydro portable export` writes the content of your site to a portable document, and `hydro portable import --dry-run` validates it before `--confirm` replaces content. Use it to move between databases or environments."),
      code('bash', "hydro portable export --path export.json --config config.yaml\nhydro portable import --path export.json --dry-run --config config.yaml"),
    ),

    page(
      'monitoring',
      'Health, metrics and diagnostics',
      "Probes for load balancers, Prometheus metrics, the doctor command and error reporting.",
      h2('Health endpoints'),
      table(
        ['Endpoint', 'Meaning', 'Use for'],
        ['`GET /healthz`', 'The process is alive.', 'Container restart policies and uptime checks.'],
        ['`GET /readyz`', 'The process is ready to serve (dependencies reachable).', 'Load balancer and rolling-deploy gates.'],
        ['`hydro healthcheck`', 'CLI probe of `/healthz` (override with `HYDRO_HEALTHCHECK_URL`).', 'Docker `HEALTHCHECK`.'],
      ),
      h2('Metrics'),
      p("`GET /metrics` exposes Prometheus metrics. Restrict it at your proxy to your monitoring network. The admin's Telemetry area shows uptime, requests, memory, database and queue statistics, and the busiest paths."),
      code('yaml', `# prometheus.yml
scrape_configs:
  - job_name: hydrogo
    metrics_path: /metrics
    static_configs:
      - targets: ['cms.internal:8080']`),
      h2('hydro doctor and validate'),
      code('bash', `hydro validate --config config.yaml     # config and database sanity
hydro doctor --config config.yaml       # deeper install checks
hydro doctor --json --config config.yaml
hydro doctor --fix --config config.yaml # apply safe automatic fixes`),
      h2('Error reporting and tracing'),
      ul(
        "Set `observability.sentry_dsn` (or `HYDRO_SENTRY_DSN`) to send errors to Sentry or GlitchTip.",
        "On the `distributed` profile, OpenTelemetry export can be enabled with `HYDRO_OTEL_ENABLED`.",
        "Every response carries `X-Request-ID`. Error bodies include it as `request_id`, and the audit trail stores it, so one id links a user report to logs.",
      ),
      h2('Background jobs'),
      p("Failed background jobs end up in a dead-letter queue. Inspect and manage it from the CLI:"),
      code('bash', `hydro jobs dead-letter list --limit 20 --json
hydro jobs dead-letter requeue --id JOB_ID --confirm`),
    ),
  ],
};
