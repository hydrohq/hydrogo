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

The session store (`src/lib/server/sessions.ts`) is in memory: one panel
process, and a restart signs everyone out. For several instances, use sticky
sessions or swap that module for Redis or a database table.

## Configuration

| Variable | Meaning |
|---|---|
| `HYDRO_API_ORIGIN` | where HydroGo runs, e.g. `http://127.0.0.1:8080` |
| `CRM_LANG` | default language, `en` or `tr` (users can switch it) |
| `CRM_SECURE_COOKIES` | `true` when the panel is served over HTTPS |

Behind a reverse proxy set `ORIGIN` (the panel's public URL) for the production
build, as for any SvelteKit adapter-node app.

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
