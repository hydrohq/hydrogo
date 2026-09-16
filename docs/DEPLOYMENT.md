# HydroGo Production Deployment Guide

Bu rehber, HydroGo projelerinin Linux, Docker ve Windows ortamlarında production ortamına güvenli ve kesintisiz biçimde dağıtılmasına yönelik standartları içerir.

---

## 1. Docker Dağıtımı

Repository kökündeki `Dockerfile` üretim için tekrarlanabilir linux/amd64 ve
linux/arm64 OCI image'ları üretir. Image içine lisans anahtarı, secret veya deployment config'i
gömülmez; çalıştırırken `/app/config.yaml` ve gerekiyorsa lisans dosyası
ayrı, erişimi kısıtlı bir volume/secret olarak bağlanmalıdır.

```powershell
pwsh -File scripts/container-lint.ps1
docker build --pull -t hydrogo:1.0.0 .
```

`scripts/container-lint.ps1` Docker daemon gerektirmeyen statik güvenlik
kapısıdır. Gerçek image build/push kanıtı, Docker/OCI registry erişimi olan
CI veya deployment ortamında ayrıca alınmalıdır.

GitHub Actions `container` işi iki platformu Linux runner üzerinde çalıştırır
ve image'ları registry'ye göndermeden gerçek `docker build` ile doğrular. Registry
push işlemi bilinçli olarak ayrı bir release/onay adımıdır.

Image healthcheck’i yalnızca loopback üzerindeki `/healthz` endpoint’ini
bounded timeout/retry ile çağırır; dış servislere veya kimlik bilgilerine
bağımlı değildir.

`scripts/secret-lint.ps1` ve `.sh`, private key ve canlı provider/token
kalıplarını yalnızca Git tarafından izlenen dosyalarda arar. Test amaçlı dummy
parolalar bu düşük false-positive’li kapının kapsamına girmez; gerçek secret’lar
her zaman deployment secret store’dan sağlanmalıdır.

### 1.1. Canonical Dockerfile

Üretim image tanımı yalnızca repository kökündeki [`Dockerfile`](../Dockerfile)
dosyasıdır. Bu dosya linux/amd64 ve linux/arm64 için multi-stage, non-root ve
bounded healthcheck içeren image üretir. Config, lisans anahtarı ve diğer
secret'lar image'a kopyalanmaz; çalışma zamanında secret store veya salt-okunur
`/app/config.yaml` mount'ı üzerinden sağlanır. Yukarıdaki `docker build` komutu
ve CI'daki `container` işi kanonik doğrulama akışıdır.

### 1.2. Docker Compose (SQLite - Kompakt Profil)

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

### 1.3. Docker Compose (PostgreSQL + Redis - Standard Profil)

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

> `sslmode=disable` yalnızca izole, yerel geliştirme ağı içindir. Standard
> profile production dağıtımında PostgreSQL TLS sertifikası ve `DB_SSLMODE`
> değeri deployment secret/config katmanından sağlanmalıdır.

---

## 2. Linux Systemd Servisi

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

Servisi etkinleştirme:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now hydrogo
sudo systemctl status hydrogo
```

---

## 3. Windows Service Kurulumu

Windows ortamında NSSM (Non-Sucking Service Manager) ile servis oluşturma:

```powershell
nssm install HydroGo "C:\HydroGo\hydro.exe" "serve --config C:\HydroGo\config.yaml"
nssm set HydroGo AppDirectory "C:\HydroGo"
nssm set HydroGo Start SERVICE_AUTO_START
nssm start HydroGo
```

---

## 4. Sağlık Kontrolü ve İzleme

- **Endpoint:** `GET /healthz`
- **Beklenen Yanıt:** `200 OK`
- **Reverse Proxy Yapılandırması (Nginx):**
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
