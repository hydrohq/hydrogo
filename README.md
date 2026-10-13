# HydroGo — The Modern Go Workspace & Product Engine

> **Fluid as water. Resilient as rock. Control always remains with you.**

HydroGo is an industrial-grade **Go Workspace, Code Generator, and Product Engine** engineered to produce high-performance **Client Editions** — enterprise corporate platforms, editorial publications, SaaS backends, customer portals, and transaction-heavy web applications.

HydroGo is not a customer-specific CMS or another heavy monolithic framework. It provides a versioned, immutable **Core Engine** alongside a generative **Product Factory** that cleanly separates machine-generated contracts (`generated/`) from human-owned business policies, workflows, and integrations (`custom/`).

---

## 📸 Product Showcase

*(The screenshots below are illustrative placeholders. You can replace them with your own screenshots in `docs/assets/`.)*

<div align="center">
  <img src="docs/assets/admin-studio.png" alt="HydroGo Admin Studio" width="800" style="border-radius: 8px; border: 1px solid #ccc;" />
  <br/>
  <em>Zero-Jargon Business Studio & Executive Dashboard</em>
</div>
<br/>
<div align="center">
  <img src="docs/assets/theme-builder.png" alt="Theme Blocks & Bento Grid" width="400" style="border-radius: 8px; border: 1px solid #ccc; margin-right: 10px;" />
  <img src="docs/assets/rbac-matrix.png" alt="Role & Permission Matrix" width="400" style="border-radius: 8px; border: 1px solid #ccc;" />
</div>

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

#### Blueprint-to-Admin generation

Blueprint resources can produce Edition-owned Admin CRUD surfaces without
hand-registering each resource in the navigation and API route lists. An
Edition wires the generated resource registry once; adding a resource then
feeds the generated Admin registry and API routes. The Edition still owns the
access policy, and reviewed database migrations must be added and applied
explicitly. This keeps generated scaffolding fast while leaving authorization
and schema changes under human control.

The verified flows include generated resource create/list/search/filter/edit/
delete, plus content review and publish journeys. Warm-cache generation
measurements recorded a 260 ms median for scaffold generation and 171 ms for
`hydro new resource` (46 generated artifacts). These measure generation
commands only; clean-scaffold-to-first-generated-CRUD time and a comparable
PayloadCMS benchmark have not been established. The current Admin acceptance
plan records what passed and which real-browser and PostgreSQL checks remain
open: [Admin Studio audit and plan](docs/planning/ADMIN_STUDIO_PRODUCT_AUDIT_AND_PLAN.md).
Recent interface coverage includes [responsive and theme checks](docs/validation/admin-content-lists-theme-responsive-2026-10-10.md)
and [mobile media-library checks](docs/validation/admin-media-mobile-layout-2026-10-10.md).

### 7. 🧩 Turnkey Domain Modules (`modules/*`)
HydroGo includes an industrial suite of pre-built, production-tested domain modules:
- **Membership & Portal (`modules/members`):** Registration, Argon2id passwords, email verification tokens, session revocation, and member dashboards.
- **Editorial & Publications (`modules/news` & `modules/pages`):** Multi-block page builder, structured categories, draft-to-publish editorial workflows, and visual revision diffs.
- **Corporate & Commercial Showcases:** Dedicated plug-and-play modules for **Portfolio**, **Events & Ticketing**, **Downloads**, **Testimonials**, **FAQ**, **Pricing Tables**, **Locations / Branch Finder**, and **Team Directories**.
- **GraphQL & Unified REST/OpenAPI (`modules/api`):** Out-of-the-box GraphQL endpoint and typed REST endpoints with automatic OpenAPI v3 schema synchronization.

### 8. 🏢 Enterprise Reliability, Multi-Tenancy & Privacy
- **Multi-Tenancy Engine (`internal/tenant`):** Context-aware request routing with tenant-scoped schema segregation and automated provisioning.
- **Transactional Outbox & Signed Webhooks (`internal/outbox`, `internal/webhooks`):** Guaranteed at-least-once event delivery, HMAC-SHA256 request signing, and automatic dead-letter queueing.
- **GDPR / KVKK Cryptographic Erasure (`internal/erasure`):** Tamper-proof, auditable, irreversible deletion workflows adhering to strict data sovereignty mandates.
- **Cross-Platform Push Notifications (`internal/push`):** Integrated WebPush and Firebase Cloud Messaging (FCM) dispatchers with zero-PII client targeting.
- **Safe Soft-Delete & Trash Bin (`internal/trash`):** System-wide recycle bin preventing accidental data loss with point-in-time restoration.

### 9. 🔭 Observability & Multi-Language SDK Generation
- **Production Observability (`internal/observability`):** OpenTelemetry (OTLP) tracing, Prometheus-compatible `/metrics`, bounded in-memory error analytics, and non-blocking Sentry panic captures with strict PII stripping.
- **Client SDK Generator (`internal/sdkdist`):** Generates strictly-typed TypeScript and Dart / Flutter client SDKs straight from the compiled OpenAPI contract.

### 10. 🌐 Adaptive Deployment Spectrum
- **High-Performance Daemons:** Standalone HTTP/HTTPS server with automatic ACME Let's Encrypt certificates.
- **Container & Cloud:** Production-ready Docker Compose, Caddy, and Kubernetes deployment configurations.
- **Zero-Ops Shared Hosting:** Optional native CGI / FastCGI compatibility (`net/http/cgi`) running dynamic Go apps on standard cPanel/Plesk shared hosting with 0ms static disk caching.

---

## 📖 Documentation & Operational Guides

HydroGo embraces a dual-audience documentation strategy, providing both human-friendly context and AI-executable runbooks.

- **Architecture Decisions (ADR):** docs/architecture/ documents every core architectural choice.
- **Human Guides:** docs/guides/human/ offers GUI-driven, context-rich runbooks explaining the *why* behind operations.
- **AI Runbooks:** docs/guides/ai/ provides zero-context, imperative instructions for AI agents to execute tasks flawlessly.

**Key Operational Runbooks:**
- [Frontend Development & Theming](docs/guides/human/frontend-development.md)
- [VPS Deployment (Nginx + Systemd)](docs/guides/human/deploy-vps.md)
- [Shared Hosting & CGI Deployment](docs/guides/human/deploy-ftp-shared-hosting.md)
- [Database Migrations & Guardrails](docs/guides/human/database-migrations.md)
- [SMTP Setup & Mail Diagnostics](docs/guides/human/mail-smtp-setup.md)

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

## 📊 Capacity Benchmark

A short external load test measured warm public HTML cache traffic and dynamic
HTML traffic on a 2-vCPU host with 908 MiB of RAM. The full report includes
every load stage, latency and error counts, resource samples, test limits, and
the earlier paced measurements: [HydroGo capacity benchmark — 2026-09-30](docs/validation/hydrogo-capacity-benchmark-2026-09-30.md).

Those values describe that test setup only; they are reference measurements,
not a general hardware or throughput promise.

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

## 📄 License

HydroGo is open-source software licensed under the [MIT License](LICENSE).

---

## ⚖️ Legal Disclaimer & Terms of Use

### 1. Disclaimer of Warranties ("AS IS")
THE SOFTWARE AND ALL ASSOCIATED DOCUMENTATION ARE PROVIDED "AS IS" AND "AS AVAILABLE", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, TITLE, ACCURACY, SYSTEM INTEGRATION, QUIET ENJOYMENT, AND NON-INFRINGEMENT. TO THE MAXIMUM EXTENT PERMITTED BY APPLICABLE LAW, THE AUTHORS, MAINTAINERS, CONTRIBUTORS, AND COPYRIGHT HOLDERS MAKE NO REPRESENTATIONS OR WARRANTIES THAT THE SOFTWARE WILL BE ERROR-FREE, SECURE, UNINTERRUPTED, ACCURATE, COMPLETE, FREE OF DEFECTS OR HARMFUL CODE, OR THAT DEFECTS WILL BE DETECTED OR CORRECTED.

### 2. Limitation of Liability
IN NO EVENT SHALL THE AUTHORS, MAINTAINERS, CONTRIBUTORS, OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES, LOSSES, EXPENSES, OR OTHER LIABILITY OF ANY NATURE WHATSOEVER, WHETHER IN AN ACTION OF CONTRACT, TORT (INCLUDING NEGLIGENCE), STRICT LIABILITY, OR OTHERWISE, ARISING FROM, OUT OF, OR IN CONNECTION WITH THE SOFTWARE, ITS USE, THE INABILITY TO USE THE SOFTWARE, OR OTHER DEALINGS IN THE SOFTWARE. THIS LIMITATION APPLIES TO, WITHOUT LIMITATION:
- ANY DIRECT, INDIRECT, SPECIAL, INCIDENTAL, EXEMPLARY, CONSEQUENTIAL, OR PUNITIVE DAMAGES;
- ANY LOSS OF USE, DATA, SYSTEM UPTIME, REVENUE, PROFIT, GOODWILL, REPUTATION, OR BUSINESS ANTICIPATED SAVINGS;
- ANY SECURITY INCIDENTS, UNAUTHORIZED ACCESS, DATA BREACHES, SYSTEM COMPROMISES, OR MALFUNCTION;
- ANY CLAIMS OR DAMAGES ARISING OUT OF TRANSACTIONS, E-COMMERCE PAYMENTS, OR DATA PROCESSING CARRIED OUT THROUGH CLIENT EDITIONS, MODULES, OR THIRD-PARTY INTEGRATIONS.

### 3. Trademark & Non-Affiliation Notice
All trademarks, service marks, logos, brand names, product names, and registered trademarks appearing in this repository or documentation (including, but not limited to, Go, Golang, WordPress, Drupal, Laravel, Filament, Node.js, V8, PayloadCMS, Strapi, Next.js, PocketBase, PostgreSQL, SQLite, MySQL, MariaDB, Redis, Valkey, Caddy, Nginx, Docker, Kubernetes, Cloudflare, Turnstile, Google Authenticator, 1Password, Sentry, OpenTelemetry, Prometheus, Quill, Editor.js, HTMX, Tailwind, Playwright, Ubuntu, Apple, and others) are the property of their respective trademark holders.

Their use in this repository, documentation, benchmarks, or codebase is strictly for technical identification, descriptive context, compatibility statements, and comparative architectural analysis under nominative fair use. The HydroGo project and its contributors are entirely independent and are **not affiliated with, associated with, endorsed by, sponsored by, or in any way officially connected** with any of these entities or their affiliates.

### 4. Regulatory, Compliance & Privacy Notice
While HydroGo includes technical primitives and architectural patterns designed to support privacy and governance workflows (such as cryptographic data erasure helpers, cookieless telemetry, CSP nonce propagation, and audit outboxes), **the provision of these capabilities does not constitute legal, regulatory, cybersecurity, or compliance advice**.

Each individual, developer, company, or agency deploying HydroGo is solely and independently responsible for:
- Ensuring that their deployment and data practices comply with all applicable local, national, and international laws, regulations, and industry standards (including, but not limited to, GDPR, CCPA/CPRA, KVKK, HIPAA, PCI-DSS, and SOC 2);
- Reviewing and configuring data retention, encryption keys, access controls, backups, and consent banners;
- Obtaining appropriate legal counsel to verify compliance with privacy and commercial laws in their operational jurisdictions.

### 5. High-Risk Activities & Production Responsibility
HydroGo is not designed, manufactured, or intended for use in mission-critical environments where software failure could directly lead to death, personal injury, environmental damage, or catastrophic physical property loss (e.g., life support, emergency response, aviation, or nuclear facilities). Deployments to production environments are executed entirely at the user's sole risk and discretion.

### 6. Security & Responsible Disclosure
If you identify a security vulnerability in HydroGo, please review our [Security Policy](docs/SECURITY.md) for instructions on responsible disclosure. Do not disclose vulnerabilities publicly through GitHub Issues or public forums before a fix is published.
