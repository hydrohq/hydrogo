# HydroGo News Site Setup & Deployment Guide

This guide describes how to scaffold, license, develop, and deploy an independent **News / Publishing** Client Edition project powered by HydroGo without bundling Core engine sources.

---

## 1. Project Scaffolding

Create a new news site project using the following command:

```powershell
hydro new haber-merkezi --preset=news --database=sqlite --profile=compact --git
```

Or run interactively:

```powershell
hydro new
# Project name: haber-merkezi
# Preset: news
# Database: sqlite
# Profile: compact
# Presentation type: integrated
# Demo content: y
# Initialize Git: y
```

Generated directory layout:
```text
haber-merkezi/
├── config.yaml          # Runtime configuration
├── hydrogo.lock         # Project identity, Core version, and feature locks
├── README.md            # Getting started guide
├── PROJECT_STATUS.md    # Status and readiness checklist
├── TASKS.md             # Project task tracker
├── LICENSE              # Client Edition license file
├── NOTICE               # Third-party notices
├── docs/                # Architecture, operations, and E2E guides
├── themes/
│   └── newsroom/        # Theme templates (theme.json, layouts/, static/)
├── custom/              # Customer-specific Go plugins and hooks
├── migrations/          # Customer-specific database migrations
├── seed/                # Initial and demo seed data
└── e2e/
    └── manifest.json    # Playwright E2E test manifest
```

> [!NOTE]
> The project directory does not contain internal HydroGo Core source code. The project is executed using the HydroGo runtime binary (`hydro`).

---

## 2. Initialization and Server Boot

1. If deploying in an environment that requires a license file, ensure `hydro.lic` is placed in the project root:
   ```powershell
   cd haber-merkezi
   hydro license status --license hydro.lic
   ```
   *(For local open-source development, run with `hydro serve --dev` or `HYDRO_DEV_MODE=1`)*

2. Apply database migrations:
   ```powershell
   hydro migrate --config config.yaml
   ```

3. Create the administrator user:
   ```powershell
   hydro create-admin --config config.yaml --email editor@haber-merkezi.test --password-stdin
   # Enter password and press Enter
   ```

4. Start the server:
   ```powershell
   hydro serve --config config.yaml
   ```

The server starts listening on `http://127.0.0.1:8080`:
- **Public Newsroom:** `http://127.0.0.1:8080/public/news/tr/html`
- **Admin Studio Panel:** `http://127.0.0.1:8080/admin`
- **Member Login:** `http://127.0.0.1:8080/member/login`
- **Health Check:** `http://127.0.0.1:8080/healthz`

---

## 3. Connecting to a Remote Git Repository

The generated project is versioned as an independent Git repository:

```powershell
cd haber-merkezi
git add .
git commit -m "Initial news project scaffold"
git remote add origin https://github.com/your-org/haber-merkezi.git
git branch -M main
git push -u origin main
```

> [!WARNING]
> Never commit private signing keys or sensitive production credentials to version control. `hydro.lic` and `config.yaml` with environment variable placeholders can be safely stored.

---

## 4. Theme Customization

Customize files under `themes/newsroom/`:
- `theme.json`: Theme metadata, slots, and asset declarations.
- `layouts/base.html`: HTML5 skeleton, meta tags, responsive viewport, and slot placements (`header`, `content`, `footer`).
- `static/theme.css`: Color palette, typography, dark mode CSS custom properties, and responsive grid rules.

---

## 5. Running E2E Tests

```powershell
npm test
```
Metadata-driven E2E tests automatically verify desktop, tablet, and mobile views for horizontal layout overflow and visual regressions.
