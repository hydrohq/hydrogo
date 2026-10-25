# HydroGo CRM panel

A small sales panel for a HydroGo project made with `hydro new --preset crm`:
a deal board, companies, contacts, activities and a task list, in English and
Turkish. HydroGo's own admin stays as the back office; this is the screen your
sales team lives in.

```bash
hydro new my-crm --preset crm --starter crm --admin-email you@example.com
cd my-crm
hydro migrate --config config.yaml
hydro serve --config config.yaml          # API on :8080
# in another terminal
cd frontend && cp .env.example .env && npm install && npm run dev   # panel on :5173
```

Sign in with the admin account you created. Demo data (optional):

```bash
HYDRO_ADMIN_EMAIL=you@example.com HYDRO_ADMIN_PASSWORD=... node scripts/seed.mjs
```

## Look and feel

A soft tinted canvas, white rounded cards and an icon rail (a tab bar on
phones); light and dark follow the system. Forms open in dialogs, tables sort
by clicking a column header, and the dashboard shows KPI cards, a stage chart
and the open-pipeline mix. Motion respects `prefers-reduced-motion`.

To make it yours, change the variables at the top of `src/app.css`:

| Variable | Used for |
|---|---|
| `--accent` | links, focus rings, chart tint |
| `--dark` | filled controls and the active navigation item |
| `--warm` | highlights, such as the hatched tallest bar |
| `--bg`, `--surface`, `--ink`, `--muted`, `--line` | canvas, cards, text and borders |
| `--s1` ... `--s8` | chart categories (kept in this order; they are checked for colour-blind separation) |

Dark-mode values live in the `prefers-color-scheme: dark` block of the same
file. Components are in `src/lib/components` (`Modal`, `Icon`, `Empty`,
`BarChart`, `DonutChart`, `CountUp`).

## How it works

The panel is a server-rendered SvelteKit app and the only thing that talks to
HydroGo. The browser never gets a HydroGo cookie or token:

1. The sign-in form posts to the panel. The panel signs in against HydroGo
   (`/auth/login`, with the authentication code when the account has MFA on),
   keeps HydroGo's session cookie and CSRF token **in server memory**, and
   gives the browser only an opaque, HttpOnly `crm_sid` cookie.
2. Every page load and form action runs on the panel's server and calls
   `/api/v1/admin/content/<type>` with the user's own session, so HydroGo
   applies that user's permissions and audit trail.
3. SvelteKit's origin check protects form posts from cross-site requests.

The session store (`src/lib/server/sessions.ts`) is in memory by default: one
panel process, and a restart signs everyone out. Set `REDIS_URL` (+
`SESSION_SECRET`) to share sessions across instances instead.

## Configuration

| Variable | Meaning |
|---|---|
| `HYDRO_API_ORIGIN` | where HydroGo runs, e.g. `http://127.0.0.1:8080` (ignored once `CRM_TENANTS` is set) |
| `CRM_LANG` | default language, `en` or `tr` (users can switch it) |
| `CRM_SECURE_COOKIES` | `true` when the panel is served over HTTPS |
| `REDIS_URL` | optional; shares sessions across panel instances instead of the per-process in-memory store |
| `SESSION_SECRET` | required when `REDIS_URL` is set; encrypts session data at rest |
| `CRM_TENANTS` | optional; serves several tenants from one deployment, resolved by Host header — see below |
| `WEBHOOK_SECRET`, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`, `NOTIFY_TO` | optional; email notifications on deal/task changes — see below |

Behind a reverse proxy set `ORIGIN` (the panel's public URL) for the production
build, as for any SvelteKit adapter-node app.

### Multiple tenants

With `CRM_TENANTS` unset, the panel is single-tenant and behaves exactly as
above. To serve several customer orgs from one deployment, set it to a JSON
array, resolved by the incoming request's Host header the same way HydroGo's
own `cfg.Tenancy.Hosts` resolves tenants server-side:

```
CRM_TENANTS=[{"slug":"acme","host":"acme.crm.example.com","hydroOrigin":"https://acme.hydro.example.com","webhookSecret":"..."}]
```

Each tenant needs a matching entry in its HydroGo deployment's own
`cfg.Tenancy.Hosts` (same host, same tenant id) — no HydroGo core changes are
needed. A session is only ever valid for the tenant it was created under.

### Email notifications

HydroGo already fires `content.created`/`content.updated`/`content.deleted`
webhook events on every collection write; the panel can turn those into
emails. Add a `webhooks:` entry to the HydroGo deployment's own config,
pointing at `https://<panel-host>/api/webhooks/hydrogo/<tenant-slug>`
(`default` for a single-tenant deployment) with topics
`content.created,content.updated,content.deleted` and a shared secret, then
set `WEBHOOK_SECRET` (or `webhookSecret` per tenant in `CRM_TENANTS`) and the
`SMTP_*`/`NOTIFY_TO` variables on the panel. The webhook payload carries no
field-level diff, so v1 can say a deal or task was created/updated/deleted,
not which field changed (e.g. not specifically "stage changed").

## Data model

Stock types from `content-types/`: companies, contacts, deals and activities,
related by id. To add a field, edit the YAML in your project, restart HydroGo,
then show the field in the matching `src/routes/` page. The deal stages are the
`stage` select in `deals.yaml` and `src/lib/pipeline.ts`; change both together.

Amounts of different currencies are never added together: the board shows one
total per currency.

## Develop

```bash
npm run check   # svelte-check
npm test        # unit tests (sessions, login handshake, pipeline, translations)
npm run build && node build
```
