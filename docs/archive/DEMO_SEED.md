# Demo Seed Scope

`seed-demo --confirm-demo` is strictly intended for local development and browser verification.
Without explicit confirmation, the command fails closed. It must never be used with production
data, real customer accounts, or live payment credentials.

Demo seed is not the default runtime surface. HydroGo's root `/` endpoint displays the backend
runtime status; the demo frontend opens at `/public/pages/{locale}/demo/html` after the seed operation.
In production deployments, demo seed must not be run, and the production frontend should be explicitly
routed via `site.home_path`. Seeded demo pages are hidden by default from public page routes.
When local browser verification is needed, start the server with `HYDRO_ENABLE_DEMO_FRONTEND=true`.

The seed makes the following flows visible and testable:

- Pages, news articles, and cover media assets.
- An example member account accessible via the public frontend: `member@hydrogo.local` / `memberpass123`.
- Idempotent records across SQLite and PostgreSQL; existing identical identities are preserved.

The example password is exclusively for local demonstration. When deploying or provisioning customer
projects, disable demo seeding or immediately change demo member credentials. Live payment providers,
real email delivery, production object storage, and analytics are not simulated by the seed.

Member accounts are strictly segregated from Admin Studio accounts; an active admin session does not
grant authorization to public member flows.
