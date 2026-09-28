# HydroGo — Workspace & Engine for Client Editions

HydroGo is a Go-based **product factory and high-performance CMS/BaaS engine**, not a single customer-specific web app. It transforms versioned blueprints, modular contracts, and domain policies into deployable, production-ready **Client Editions** for corporate websites, editorial publications, portals, SaaS backends, and transaction-heavy business products.

The Core remains generic and versioned; specific customer workflows, authorization rules, and presentation layers live strictly within an Edition's `custom/` boundary.

---

## ⚡ Why HydroGo? The Paradigm Shift

Traditional CMS and BaaS stacks force painful trade-offs:
- **WordPress / PHP:** Requires 150+ disk reads per request, consumes 50–100MB RAM per worker, suffers from plugin vulnerabilities, and exposes all plain-text PHP source code to anyone with FTP access.
- **Node.js / PayloadCMS / Strapi:** Demands high-memory VPS/containers ($15–30/month), suffers 1.5–3s cold-starts, requires heavy JavaScript builds, and cannot run on affordable shared hosting.
- **Laravel / Filament:** Heavy dependency tree, high memory footprint, requires persistent PHP-FPM daemons.
- **PocketBase:** Excellent SQLite BaaS, but lacks a server-rendered (SSR) typed Templ design system, agency client-handover UI, and native CGI shared-hosting deployment.

**HydroGo bridges this gap:** A single compiled binary (~12–16MB) running with **2–4ms cold-start and 8–15MB RAM**, fully dynamic SSR with type-safe Go/Templ, and native CGI support for **$10–15/year cPanel/Plesk shared hosting**, while preserving complete intellectual property code secrecy.

---

## 📊 Feature Comparison Matrix

| Feature / Capability | WordPress | PayloadCMS | Laravel Filament | PocketBase | **HydroGo** |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **Runtime Language** | PHP (Interpreted) | TypeScript / Node.js | PHP (Interpreted) | Go (Compiled) | **Pure Go (Compiled)** |
| **$10/yr cPanel / Shared Hosting** | ⚠️ Slow / Heavy | ❌ Impossible | ⚠️ High Memory | ❌ No native CGI | **✅ Native CGI / FastCGI** |
| **Cold Start / Response Time (TTFB)** | 150ms – 600ms | 800ms – 2500ms | 100ms – 350ms | 5ms – 20ms | **2ms – 10ms (0ms Static Cache)** |
| **Memory Footprint (Per Request)** | 40MB – 120MB | 120MB – 300MB | 30MB – 80MB | 15MB – 30MB | **8MB – 15MB (Compact Profile)** |
| **Code Secrecy & IP Protection** | ❌ Plaintext PHP | ❌ Plaintext JS | ❌ Plaintext PHP | ✅ Compiled Binary | **✅ Stripped Machine Code ELF** |
| **External Dependencies** | MySQL, Apache | Node.js, MongoDB/PG | PHP-FPM, MySQL | None (SQLite) | **Zero (In-Process SQLite)** |
| **Frontend SSR Architecture** | PHP Spaghetti / Theme | React / Next.js SPA | Blade / Livewire | Client-Side SPA / API | **Type-Safe Templ + HTMX** |
| **AI SEO (`/llms.txt` & OG Generator)**| ❌ Requires Plugin | ❌ Custom Code | ❌ Custom Code | ❌ None | **✅ Built-in Pure Go Engine** |
| **SMTP Crash Shield (Mail Spool)**| ❌ Lost Emails | ❌ None | ⚠️ Redis Queue | ❌ None | **✅ SQLite Spool + MX Fallback** |
| **Client Handoff UX (Zero-Jargon)** | ❌ Technical Clutter | ⚠️ Dev-Centric | ⚠️ Admin-Centric | ⚠️ DB-Viewer Centric | **✅ Business Owner Studio** |

---

## 🏛️ Architecture at a Glance

```text
Blueprint / Edition Manifest / Module Contracts
                         │
                         ▼
                 generated/ Artifacts
                         │      custom/ Policy + Workflow
                         └──────────────┐
                                        ▼
HTTP/JSON/Templ → Application Service → Repository Contract → SQLite / Postgres / MySQL
                                        │
                         Audit · Jobs · Cache · Media · Observability
```

---

## 🚀 Key Modules & Capabilities

### 1. 🌐 Hosting & Runtime Revolution ($10/Year Shared Hosting)
- **Native CGI & FastCGI (`net/http/cgi` & `net/http/fcgi`):** Runs on the cheapest cPanel/Plesk shared hosting plans as a single binary with zero VPS maintenance overhead.
- **Automatic Static HTML Disk Caching (`.htaccess` Passthrough):** Public pages rendered for unauthenticated visitors are cached to disk; subsequent requests are served directly by Apache (**0ms TTFB / 0 CPU cycles**).
- **Lazy Web Cron:** Runs background maintenance (session grooming, news unpublishing, webhook delivery) at the tail of incoming HTTP requests when persistent cron daemons are unavailable.
- **Adaptive SQLite Profiles:** Ultra-light *Compact Profile* (2MB cache) for shared hosting, high-throughput *VPS Profile* (64MB cache, mmap) for dedicated instances.

### 2. 🎨 Type-Safe Templ Design System (`modules/designsystem`)
- **Bento Grid & Block Layout Engine:** Modern asymmetric block arrangements inspired by Apple and Linear.
- **Interactive Components:** FAQ Accordion (with automated `Schema.org FAQPage` JSON-LD), Before/After Image Comparison Slider, Lightweight Video Player, Image Hotspots, Animated Number Counters (`IntersectionObserver`).
- **Micro-Interactions:** Copy-to-Clipboard (`ui.CopyButton`), Form Success Confetti, Smart Sticky Header (auto-hide on scroll down, reveal on scroll up), Pure CSS Masonry Gallery.

### 3. 📝 Form Engine, Lead CRM & Resilient Mailer (`modules/forms` & `modules/mailer`)
- **Form Builder & Inbox:** Dynamic field definitions, honeypots, and Cloudflare Turnstile bot deterrence.
- **Double-Submission Lock (Idempotency):** Prevents duplicate form submissions during network stutter.
- **Automatic Autoresponders:** HTML confirmation emails with dynamic tags (`{{name}}`, `{{ticket_code}}`).
- **Email Typo Suggestion:** Live micro-UX catching common domain mistakes (`@gmai.com`, `@hotmial.com`).
- **Durable SQLite Mail Spooler:** Spools outgoing emails with exponential backoff (`5m`, `15m`, `1h`, `6h`, `24h`) so no customer inquiry is ever lost when an SMTP server is unreachable.
- **Zero-Config MX Fallback:** Directly delivers critical emails to destination DNS MX records if SMTP credentials are omitted.

### 4. 📈 SEO, AI Search & Privacy-First Analytics (`modules/seo` & `modules/analytics`)
- **Pure Go Dynamic OpenGraph (OG) Generator:** Generates branded 1200x630 social share cards on-the-fly without Chromium or Node.js (`image/draw` + embedded fonts).
- **AI Crawler Index (`/llms.txt` & `/llms-full.txt`):** Markdown sitemap tailored for LLMs like ChatGPT, Claude, and Perplexity.
- **Zero-Cookie GDPR Analytics:** Privacy-compliant, cookie-free server-side pageview, visitor, and referrer analytics.
- **Dynamic Sitemap Splitter:** Handles 50,000+ URLs with automated `sitemap-index.xml` and section chunking.
- **Performance Middleware:** ETag 304 Not Modified, Pure Go Brotli & Gzip compression, `sync.Pool` zero-allocation rendering buffers.

### 5. 🛡️ Bulletproof Security & Resilience (`internal/security`)
- **Tarpit Progressive Sleep:** Progressively delays failed login attempts (1s -> 2s -> 4s -> 8s) to stall brute-force attackers without consuming server CPU.
- **TOTP Two-Factor Authentication (2FA):** Google Authenticator / 1Password compatible RFC 6238 QR-code verification.
- **Instant IP / CIDR Ban (`hydro ban`):** Memory- and database-backed immediate 403 Forbidden firewall.
- **SQLite Startup WAL Recovery:** Automatically heals orphaned `.db-wal` locks after unexpected system crashes.
- **Disk Quota Guard:** Safely throttles uploads when disk usage exceeds 90% to prevent database corruption.
- **Single-Command Rollback (`hydro rollback`):** Reverts to the snapshot taken immediately before the last migration.
- **Immutable Audit Log:** Tamper-evident record of all administrative mutations and logins.

### 6. 💼 Business Owner Admin Studio (`internal/adminstudio`)
- **Zero Technical Jargon:** Clean client-facing UI that hides internal database identifiers and technical stack traces (`docs/INFORMATION_HIERARCHY.md`).
- **Visual Translation Matrix (i18n):** Search, preview, and update system and custom translations in real time.
- **Live Visual Diff:** Color-coded green/red comparison across content revisions.
- **Floating Bulk Action Toolbar:** Multi-select table actions for batch export, marking as read, or deletion.
- **CSV / XLSX Import Wizard:** Automated column mapping and validation for customer data migration.
- **Monday Executive Digest:** Weekly summary email delivered to stakeholders every Monday at 09:00.
- **Agency Support Card & Video Guide Tab:** In-panel agency contact card and embedded video tutorials for seamless client onboarding.

---

## 🛠️ Unified Cross-Platform CLI

HydroGo commands use the **exact same syntax** on Linux, macOS, and Windows:

```bash
# Global Installation
go install ./cmd/hydro

# Create a New Project (Presets: corporate, news, agency, saas, minimal)
hydro new my-client-site --preset=corporate
cd my-client-site
hydro project check

# Development Mode (Hot-Reloading Server)
hydro dev

# Database & Diagnostics
hydro migrate                         # Apply pending migrations
hydro doctor                          # Inspect system health
hydro doctor --fix                    # Repair SQLite permissions and locks
hydro rollback                        # Roll back to pre-migration snapshot

# Production Server
hydro serve                           # Standalone HTTP server
hydro serve --cgi                     # cPanel / Plesk CGI mode
hydro ban 198.51.100.4                # Block malicious IP
hydro clean --demo                    # Purge mock/demo dataset
```

---

## 📦 Deployment Options

### 1. Shared Hosting (cPanel / Plesk — $10/Year)
```bash
make build-cgi
```
Upload the compiled `hydro.cgi` binary and `deploy/hosting/.htaccess` file to `public_html/` via FTP/SFTP. **Your dynamic site and management studio are live instantly!**

### 2. Standalone VPS / Container (Docker / Caddy)
```bash
docker compose -f deploy/docker-compose.prod.yml up -d
```

---

## 🧪 Testing & Verification

Run the full uncached test suite:

```bash
go test ./...
```

Run race detector:

```bash
go test -race ./...
```

---

## 📄 License & Intellectual Property

HydroGo is proprietary software. Licensing and commercial usage terms are defined in [`PROPRIETARY_NOTICE.md`](PROPRIETARY_NOTICE.md) and [`docs/COMMERCIAL_LICENSING.md`](docs/COMMERCIAL_LICENSING.md).
