# HydroGo Production Deployment Guide

This guide describes operational standards for securely and reliably deploying HydroGo projects
across Linux, Docker, and Windows environments.

---

## 1. Docker Deployment

The canonical `Dockerfile` at the repository root generates reproducible linux/amd64 and linux/arm64
OCI images. Secret keys, credentials, and deployment-specific configurations are never baked into the image;
at runtime, `/app/config.yaml` and any required files are mounted as segregated, access-restricted volumes or secrets.

```powershell
pwsh -File scripts/container-lint.ps1
docker build --pull -t hydrogo:1.0.0 .
```

`scripts/container-lint.ps1` serves as a static security gate requiring no Docker daemon. Real image build
and registry push verifications must be performed within CI or deployment environments equipped with OCI registry credentials.

The GitHub Actions `container` workflow builds for both platforms on Linux runners, verifying images with
actual `docker build` executions without pushing to public registries. Pushing to a registry is an explicit, reviewed release step.

Image healthchecks query the `/healthz` endpoint on loopback only with bounded timeouts and retries,
maintaining zero dependencies on external third-party services or credentials.

`scripts/secret-lint.ps1` and `.sh` scan for private keys and live provider credentials strictly across
Git-tracked files. Test fixture passwords are excluded from this low-false-positive gate; actual production
secrets must always be injected via runtime secret management.

### 1.1. Canonical Dockerfile

The official production image definition is located at [`Dockerfile`](../Dockerfile) in the repository root.
It builds multi-stage, non-root images with bounded healthchecks for linux/amd64 and linux/arm64. Config, license
files, and secrets are never copied into the image; they are supplied at runtime via secret stores or read-only
volume mounts to `/app/config.yaml`. The `docker build` command above and the CI `container` workflow form the canonical verification loop.

### 1.2. Docker Compose (SQLite - Compact Profile)

```yaml
version: "3.8"

services:
  hydrogo:
    image: my-hydrogo-app:1.0.0
    container_name: hydrogo_app
    restart: unless-stopped
    init: true
    read_only: true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=64m
    ports:
      - "8080:8080"
    environment:
      - HYDRO_HTTP_ADDRESS=0.0.0.0:8080
      - HYDRO_CAPACITY_PROFILE=compact
      - HYDRO_DATABASE_DIALECT=sqlite
      - HYDRO_DATABASE_DSN=/app/data/prod.sqlite
      - HYDRO_MEDIA_DIR=/app/data/media
      - HYDRO_MEDIA_FFMPEG=ffmpeg
    volumes:
      - hydro_data:/app/data
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://127.0.0.1:8080/healthz"]
      interval: 10s
      timeout: 3s
      retries: 3

volumes:
  hydro_data:
```

### 1.3. Docker Compose (PostgreSQL + Redis - Standard Profile)

```yaml
version: "3.8"

services:
  db:
    image: postgres:16-alpine
    container_name: hydrogo_pg
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
    environment:
      POSTGRES_DB: hydrogo
      POSTGRES_USER: hydro
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U hydro -d hydrogo"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: hydrogo_redis
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5

  hydrogo:
    image: my-hydrogo-app:1.0.0
    container_name: hydrogo_app
    restart: unless-stopped
    init: true
    read_only: true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=64m
    depends_on:
      - db
      - redis
    ports:
      - "8080:8080"
    environment:
      - HYDRO_HTTP_ADDRESS=0.0.0.0:8080
      - HYDRO_CAPACITY_PROFILE=standard
      - HYDRO_DATABASE_DIALECT=postgres
      - HYDRO_DATABASE_DSN=postgres://hydro:${DB_PASSWORD}@db:5432/hydrogo?sslmode=${DB_SSLMODE:-require}
      - HYDRO_REDIS_URL=redis://redis:6379/0
      - HYDRO_MEDIA_DIR=/app/data/media
      # Optional S3-compatible media profile (inject credentials via secrets):
      # - HYDRO_MEDIA_STORAGE_DRIVER=s3
      # - HYDRO_MEDIA_S3_ENDPOINT=https://objects.example.test
      # - HYDRO_MEDIA_S3_BUCKET=hydrogo-media
      # - HYDRO_MEDIA_S3_REGION=auto
      # - HYDRO_MEDIA_S3_USE_SSL=true
      # - HYDRO_MEDIA_S3_FORCE_PATH_STYLE=false
      # - HYDRO_MEDIA_CDN_BASE_URL=https://cdn.example.test/media
    volumes:
      - media_data:/app/data/media
      - ./config.yaml:/app/config.yaml:ro
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://127.0.0.1:8080/healthz"]
      interval: 10s
      timeout: 3s
      retries: 5

volumes:
  pg_data:
  media_data:
```

> `sslmode=disable` is strictly for isolated local development networks. In production Standard
> profile deployments, PostgreSQL TLS certificates and verified `DB_SSLMODE` configurations must
> be supplied via secret stores.

---

## 2. Linux Systemd Service

`/etc/systemd/system/hydrogo.service`:

```ini
[Unit]
Description=HydroGo Platform Service
After=network.target postgresql.service

[Service]
Type=simple
User=hydro
Group=hydro
WorkingDirectory=/opt/hydrogo
ExecStartPre=/opt/hydrogo/hydro migrate --config /opt/hydrogo/config.yaml
ExecStart=/opt/hydrogo/hydro serve --config /opt/hydrogo/config.yaml
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536
EnvironmentFile=/etc/hydrogo/environment

[Install]
WantedBy=multi-user.target
```

Enabling and managing the service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now hydrogo
sudo systemctl status hydrogo
```

---

## 3. Windows Service Setup

Configuring as a Windows Service using NSSM (Non-Sucking Service Manager):

```powershell
nssm install HydroGo "C:\HydroGo\hydro.exe" "serve --config C:\HydroGo\config.yaml"
nssm set HydroGo AppDirectory "C:\HydroGo"
nssm set HydroGo Start SERVICE_AUTO_START
nssm start HydroGo
```

---

## 4. Health Checks and Reverse Proxy

- **Endpoint:** `GET /healthz`
- **Expected Status:** `200 OK`
- **Reverse Proxy Configuration (Nginx):**
  ```nginx
  server {
      listen 80;
      server_name news.example.com;

      location / {
          proxy_pass http://127.0.0.1:8080;
          proxy_set_header Host $host;
          proxy_set_header X-Real-IP $remote_addr;
          proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
          proxy_set_header X-Forwarded-Proto $scheme;
      }
  }
  ```

## 5. Admin Studio Capacity Notes

- When `admin_support` is disabled, support card configurations do not spawn background workers or allocate auxiliary resources. To enable, configure `admin_support.enabled` and at least one contact channel.
- Page imports process a maximum bounded limit of 5 MiB, 5,000 rows, and 100 columns per request.
- Notifications and digest preferences reside in the primary database; no separate cache tier is required. If the content module is inactive, notification migrations are omitted. Retention boundaries must be configured prior to enabling high-volume notification producers.
- Weekly digests are disabled by default. Activated per user preference, they are scheduled via the existing mail-spool worker when Forms, Pages, and SMTP are enabled. Digest emails enqueue into the standard SQLite mail spool and reuse standard queue retry policies.
