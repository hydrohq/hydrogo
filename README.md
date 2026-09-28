# HydroGo — The Modern Go Workspace & Product Engine

> **Fluid as water. Resilient as rock. Control always remains with you.**

HydroGo is an industrial-grade **Go Workspace, Code Generator, and Product Engine** engineered to produce high-performance **Client Editions** — enterprise corporate platforms, editorial publications, SaaS backends, customer portals, and transaction-heavy web applications.

HydroGo is not a customer-specific CMS or another heavy monolithic framework. It provides a versioned, immutable **Core Engine** alongside a generative **Product Factory** that cleanly separates machine-generated contracts (`generated/`) from human-owned business policies, workflows, and integrations (`custom/`).

---

## ⚡ The HydroGo Philosophy & Vision

Modern web development is burdened by two extremes:
1. **Fragile Monoliths (WordPress, Drupal, Laravel):** Heavy PHP interpreter overhead, high memory footprint (50–150MB/worker), constant plugin security vulnerabilities, and plain-text source code exposed to server operators.
2. **JavaScript Framework Sprawl (Node.js, PayloadCMS, Strapi, Next.js):** 1.5–3s cold-starts, bloated `node_modules` trees, complex build pipelines, heavy container memory requirements ($20–50/mo minimums), and fragile runtime dependencies.

### The HydroGo Paradigm Shift:
- **Pure Go Velocity & Efficiency:** A single stripped binary (~12–16MB) executing with **2–4ms cold-start and 8–15MB RAM**, delivering sub-10ms server-rendered HTML at thousands of requests per second.
- **Type-Safe Full-Stack SSR:** Powered by **Go + Templ + HTMX**, eliminating frontend JavaScript runtime fatigue while maintaining instant reactive UI with zero layout shifts.
- **Pure Go Media & Assets:** Zero external runtime dependencies (no Chromium, Node.js, FFmpeg, or Python). BlurHash, WebP conversion, focal-point smart cropping, watermarking, and dynamic OpenGraph social card generation are all handled in pure Go.
- **Dual-Brain Architecture (`generated/` vs `custom/`):** Code generators produce typed models, repositories, and route scaffolding. Developers write custom policy and integrations in human-owned boundaries. Upgrading Core never destroys your custom business logic.
- **True Data Sovereignty & Portability:** You own your binaries, builds, and data. Certified across **SQLite, PostgreSQL, and MySQL/MariaDB** with automated migrations and single-command rollback.
- **Zero Technical Noise for Business Clients:** An executive Admin Studio designed around strict information hierarchy rules (`docs/INFORMATION_HIERARCHY.md`) — no UUIDs, internal error codes, or technical clutter exposed to end users.
- **Deploy Anywhere Across the Spectrum:** Seamlessly scales from enterprise container clusters and bare-metal VPS down to native CGI edge environments.

---

## 📊 Competitive Architecture Matrix

| Capability / Metric | WordPress | PayloadCMS / Strapi | Laravel Filament | PocketBase | **HydroGo** |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **Core Runtime** | PHP (Interpreted) | Node.js (V8) | PHP (Interpreted) | Go (Compiled) | **Pure Go (Compiled)** |
| **Response Latency (TTFB)** | 150ms – 600ms | 800ms – 2500ms | 100ms – 350ms | 5ms – 20ms | **2ms – 10ms (0ms with Disk Cache)** |
| **Memory Footprint** | 50MB – 150MB | 150MB – 400MB | 40MB – 100MB | 15MB – 30MB | **8MB – 15MB (Compact Profile)** |
| **Frontend SSR Engine** | PHP / Classic Theme | React / Next.js SPA | Blade / Livewire | Client-Side SPA / API | **Type-Safe Templ + HTMX** |
| **Source Code & IP Protection** | ❌ Plaintext PHP | ❌ Plaintext JS | ❌ Plaintext PHP | ✅ Compiled Binary | **✅ Stripped Machine Code Binary** |
| **External Heavy Tools** | Node, Apache, MySQL | Node.js, Chromium, PG | PHP-FPM, Redis, MySQL | None (SQLite) | **Zero (In-Process Pure Go)** |
| **OG Image & Social Generation** | ❌ Requires Heavy Plugins | ❌ Node/Puppeteer Required | ❌ PHP GD / Extension | ❌ None | **✅ Built-in Pure Go Engine** |
| **LLM / AI Search Indexing** | ❌ Plugin-reliant | ❌ Custom code required | ❌ Custom code required | ❌ None | **✅ Built-in `/llms.txt` Engine** |
| **Email Spool & Crash Protection** | ❌ Lost on SMTP failure | ❌ None | ⚠️ Redis Queue Required | ❌ None | **✅ SQLite Spool + MX Direct Fallback** |
| **Client-Facing Admin UX** | ❌ Cluttered / Technical | ⚠️ Dev/Schema-Centric | ⚠️ Admin-Centric | ⚠️ Database-Viewer Centric | **✅ Zero-Jargon Business Studio** |
| **Multi-Database Parity** | ❌ MySQL only | ⚠️ Mongo / PG | ⚠️ MySQL / PG | ❌ SQLite only | **✅ SQLite, PostgreSQL, MySQL Certified** |

---

## 🏛️ Dual-Brain Architecture

```text
               Blueprint Manifest & Domain Declarations
                                  │
                                  ▼
                   Code Generators (`internal/generator`)
                                  │
         ┌────────────────────────┴────────────────────────┐
         ▼                                                 ▼
generated/ Machine-Owned                           custom/ Human-Owned
├── models.go (Typed Domain Entities)              ├── policies.go (Business Rules & ACL)
├── repository/ (Certified SQL Store)              ├── workflows.go (Custom Domain Logic)
└── transport/ (HTTP / JSON / Templ)               └── extensions.go (3rd-Party Integrations)
         │                                                 │
         └────────────────────────┬────────────────────────┘
                                  ▼
         Application Services & Provider Contracts
                                  │
     ┌────────────────────────────┼────────────────────────────┐
     ▼                            ▼                            ▼
SQLite Engine             PostgreSQL Adapter           MySQL / MariaDB
(Single-File / Zero-Ops)   (Enterprise High-Scale)      (Certified Cloud SQL)
```

---

## 💎 Core Engine Capabilities

### 1. 🎨 Type-Safe Design System & Blocks (`modules/designsystem`)
- **Modern Bento Grid Engine:** Responsive, asymmetric block layouts designed for high-conversion corporate showcases.
- **Interactive UI Components:** FAQ Accordion (with automated `Schema.org FAQPage` JSON-LD), Before/After Comparison Slider, Lightbox, Image Hotspots, and Animated Number Counters (`IntersectionObserver`).
- **Zero-JS Micro-Interactions:** Smart Sticky Header (auto-hides on scroll down, reveals on scroll up), Copy-to-Clipboard (`ui.CopyButton`), Confetti celebrations, and pure CSS Masonry galleries.

### 2. 📝 Form Engine, Lead CRM & Resilient Mailer (`modules/forms` & `internal/mailer`)
- **Dynamic Form Builder & Inbox:** Custom field schema, automated honeypots, and Cloudflare Turnstile bot protection.
- **Idempotent Submission Lock:** Prevents accidental double submissions during transient network delays.
- **Automated Client Autoresponders:** Rich HTML transactional emails with dynamic tags (`{{name}}`, `{{ticket_code}}`).
- **Typo-Correction UX:** Real-time domain correction suggestions (`@gmai.com` -> `@gmail.com`).
- **Durable SQLite Mail Spooler:** Outgoing email queue with exponential backoff (`5m`, `15m`, `1h`, `6h`, `24h`) guaranteeing zero lead loss during SMTP provider outages.
- **Emergency DNS MX Fallback:** Directly queries recipient MX records and delivers mail even if third-party SMTP services fail entirely.

### 3. 🖼️ Pure Go Media Pipeline (`internal/media`)
- **Zero-Dependency Image Processing:** On-the-fly resizing, WebP conversion, and compression using standard Go libraries.
- **BlurHash & LQIP:** Automatic low-quality image placeholder generation for instant progressive image loading.
- **Focal-Point Smart Cropping:** Intelligent coordinate-based cropping preserving critical subjects across all viewports.
- **Automated PWA Icon Pack:** Generates multi-resolution icons (`192px`, `512px`, `apple-touch-icon.png`, `favicon.ico`) and `manifest.webmanifest` automatically from a single uploaded logo.
- **Asset Watermarking & Upload Guards:** Tenant-scoped watermarking, SVG script sanitization, and MIME verification.

### 4. 📈 AI SEO, Discovery & Privacy-First Analytics (`modules/seo` & `internal/analytics`)
- **Pure Go Dynamic OpenGraph (OG) Generator:** Instant 1200x630 branded social cards generated via `image/draw` and embedded fonts without Chromium or Node.js.
- **AI Crawler Indexes (`/llms.txt` & `/llms-full.txt`):** Structured markdown documentation designed specifically for ChatGPT, Claude, and Perplexity ingestion.
- **Zero-Cookie Server Analytics:** GDPR/KVKK-compliant, cookieless telemetry tracking pageviews, referrers, and user journeys.
- **Dynamic 50,000+ URL Sitemap Splitter:** Automated category chunking and `sitemap-index.xml` generation.
- **LocalBusiness & Article JSON-LD:** Rich snippet metadata dynamically injected into page heads for maximum search engine visibility.

### 5. 🛡️ Bulletproof Security & Self-Healing Runtime (`internal/security`)
- **Tarpit Progressive Sleep:** Exponentially stalls brute-force login attacks (1s -> 2s -> 4s -> 8s) without burning server CPU.
- **TOTP Two-Factor Authentication (2FA):** Standard RFC 6238 QR-code verification compatible with Google Authenticator and 1Password.
- **Instant IP & CIDR Firewall (`hydro ban`):** Memory- and database-backed immediate 403 Forbidden blocking.
- **Startup WAL Self-Healing:** Automatically cleans and recovers orphaned `.db-wal` locks after ungraceful server terminations.
- **Disk Quota Guard:** Gracefully halts file uploads when storage hits 90% capacity, safeguarding database integrity.
- **Single-Command Rollback (`hydro rollback`):** Instantly reverts to the pre-migration snapshot in disaster recovery scenarios.

### 6. 💼 Business Owner Admin Studio (`internal/adminstudio`)
- **Zero Technical Jargon:** Executive-friendly workspace hiding internal database IDs, technical stack traces, and developer noise (`docs/INFORMATION_HIERARCHY.md`).
- **Visual Translation Matrix (i18n):** Real-time search, inline translation editing, and database-backed localization overrides.
- **Live Visual Revision Diff:** Side-by-side green/red visual diff tracking every change across document revisions.
- **Floating Bulk Action Toolbar:** Multi-select table operations for instant Excel export, status updates, and batch archiving.
- **Monday Morning Executive Digest:** Automated weekly KPI and form submission digest delivered to business stakeholders every Monday at 09:00.
- **Agency Support Card & Video Guide Tab:** In-app agency communication box and embedded video training player for smooth client handoff.

### 7. 🌐 Adaptive Deployment Spectrum
- **High-Performance Daemons:** Standalone HTTP/HTTPS server with automatic ACME Let's Encrypt certificates.
- **Container & Cloud:** Production-ready Docker Compose, Caddy, and Kubernetes deployment configurations.
- **Zero-Ops Shared Hosting:** Optional native CGI / FastCGI compatibility (`net/http/cgi`) running dynamic Go apps on standard cPanel/Plesk shared hosting with 0ms static disk caching.

---

## 🛠️ Unified Cross-Platform CLI

HydroGo provides a universal CLI that behaves identically on Linux, macOS, and Windows:

```bash
# Install the HydroGo CLI globally
go install ./cmd/hydro

# Scaffold a new Client Edition (Presets: corporate, news, agency, saas, minimal)
hydro new my-client-edition --preset=corporate
cd my-client-edition
hydro project check

# Start Development Server with Hot-Reloading
hydro dev

# Database Management & Diagnostics
hydro migrate                         # Apply pending certified migrations
hydro doctor                          # Inspect runtime and schema health
hydro doctor --fix                    # Repair SQLite permissions, WAL locks, and schema
hydro rollback                        # Roll back to pre-migration snapshot

# Production Server Execution
hydro serve                           # Run standalone high-performance HTTP server
hydro serve --acme                    # Run with automated Let's Encrypt SSL
hydro ban 198.51.100.4                # Block malicious IP or CIDR block
hydro clean --demo                    # Purge mock/demo dataset
```

---

## 📦 Deployment Options

### 1. Standalone VPS / Container (Docker / Caddy)
```bash
docker compose -f deploy/docker-compose.prod.yml up -d
```

### 2. Standard Shared Hosting (cPanel / Plesk)
```bash
make build-cgi
```
Deploy the compiled `hydro.cgi` and `deploy/hosting/.htaccess` file to `public_html/`. The entire dynamic platform and Admin Studio run out-of-the-box with zero VPS management overhead.

---

## 🧪 Testing & Verification

HydroGo adheres to strict verification discipline across all supported database adapters:

```bash
# Run full unit and integration test suite
go test ./...

# Run uncached tests
go test -count=1 ./...

# Run concurrency and race detection
go test -race ./...
```

---

## 📄 License & Commercial Terms

HydroGo is proprietary software. Commercial licensing terms, distribution rights, and client edition permissions are detailed in [`PROPRIETARY_NOTICE.md`](PROPRIETARY_NOTICE.md) and [`docs/COMMERCIAL_LICENSING.md`](docs/COMMERCIAL_LICENSING.md).
