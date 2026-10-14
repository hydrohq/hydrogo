# Operational guides

Step-by-step runbooks for recurring operational tasks (deployment, DNS, mail,
etc.) that aren't architecture decisions (those are ADRs in
`docs/architecture/`) and aren't one-off validation evidence (that's
`docs/validation/`). A guide documents a repeatable *how*, written from a
real, completed run — not a hypothetical.

## Two audiences, same topic, paired filenames

Every guide exists in two versions with the **same filename** under
different folders:

- **`ai/<topic>.md`** — written for an agent (Claude Code or similar) to
  execute directly. Imperative, exact commands with placeholders, explicit
  decision points ("if X, do Y"), a troubleshooting table keyed by literal
  error text. Assumes tool access (shell, FTP/SSH, etc.) and zero prior
  context beyond what's in the file. No narrative, no "why" beyond what's
  needed to make a correct decision.
- **`human/<topic>.md`** — written for a person doing the same task, usually
  through a GUI (Plesk, Cloudflare dashboard, etc.) rather than raw commands.
  Friendlier tone, explains *why* a step matters, points to where things are
  in the panel, can defer to the `ai/` version for exact CLI syntax rather
  than duplicating it.

Keep the two in sync when the underlying process changes — if you fix a step
in one, check whether the other needs the same fix.

## Adding a new guide

1. Pick a short, kebab-case topic name (e.g. `deploy-ftp-shared-hosting`,
   `dns-cloudflare-setup`, `mail-resend-setup`).
2. Write `ai/<topic>.md` and `human/<topic>.md`.
3. Only write from a task you (or the agent) actually completed and
   verified — a guide that was never run against the real thing tends to be
   wrong in a way nobody notices until it breaks someone's deploy. If a step
   is genuinely untested, say so explicitly rather than presenting it as
   proven.
4. Redact real secrets (passwords, API keys, tokens) — use placeholders like
   `<db_password>`. Guides live in the repo and get pushed to GitHub.

## Current guides

| Topic | What it covers |
|---|---|
| `deploy-ftp-shared-hosting` | Building, FTP-uploading, and verifying a HydroGo deploy on CGI shared hosting (Plesk or similar), including the SSH-less migration/admin-bootstrap trick and the full troubleshooting matrix from a real first deployment. |
| `local-development-setup` | Running `hydro run` in a Core checkout: dev proxy ports, auto-migration, Templ auto-watch, demo seeding, project layout orientation. |
| `admin-panel-access` | Creating the first admin account (never automatic), logging in, roles, MFA. |
| `frontend-development` | Public theme and Admin Studio build steps, cache-busting/token-lint checks, and a UI quality workflow using design tokens, typed Templ components, Tailwind, complete interaction states, and responsive visual review. Includes component-gallery recommendations and the Edition boundary for a separate React/Next.js frontend. |
| `deploy-vps` | Deploying `hydro serve` to a VPS behind nginx + systemd — verified end-to-end on a real Ubuntu box, including the `public_origin`/`Host`-header requirement and cloud-provider-firewall-vs-SSH distinction discovered while testing. |
| `redeploy-live-site` | The build→upload→restart/verify loop for shipping a code change to an already-live deployment (shared hosting or VPS), not a first-time setup. |
| `mail-smtp-setup` | SMTP relay config (Resend as the worked example), required DNS records, the single-SPF-record rule, and a spam-placement diagnostic checklist. |
| `database-migrations` | Migration file anatomy, the checksum/migrationguard requirement, and the MySQL-dialect-forgotten bug class found across 20+ files this session — read before hand-writing dialect-aware SQL. |
| `dns-cloudflare-setup` | Pointing a domain at Cloudflare, propagation troubleshooting, and the Full(strict) SSL mode requirement. |
| `reading-production-logs` | Fetching and interpreting shared-hosting error logs over FTP, including why a cache hit produces no log line at all. |
