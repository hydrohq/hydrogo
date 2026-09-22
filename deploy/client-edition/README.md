# Client Edition deployment golden path

This directory is the reference deployment for a generated Client Edition,
not a second application architecture. The Edition imports HydroGo contracts,
keeps customer policy/workflow code in `custom/`, and packages the reviewed
Core/module result as one non-root static binary. PostgreSQL, media, secrets
and Caddy state are external persistent resources.

The Compose topology is intentionally explicit:

```text
Caddy (TLS/reverse proxy) → HydroGo binary → PostgreSQL
                                   └──────→ optional Redis/S3/OTLP providers
```

No service runs migrations implicitly. The operator applies the migration
registry, verifies health/readiness, and retains the previous immutable image
for blue/green rollback.

1. Copy `.env.example` to `.env`, replace every placeholder and provide the
   DSN through the deployment secret store (never commit it).
2. Run `hydro migrate --database postgres` as an operator before starting the
   new image. Runtime startup never executes DDL.
3. Run `docker compose --env-file .env up -d --build` and verify `/healthz`
   and `/readyz` through Caddy.
4. For rollback, redeploy the previous immutable image digest, then run
   `hydro migrate status`; never reverse a migration by deleting tables.

The app is non-root and the binary is the only application artifact. Caddy is
the TLS/reverse-proxy boundary; Dokploy can import this Compose file directly.
