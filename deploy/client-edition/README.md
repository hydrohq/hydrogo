# Client Edition deployment golden path

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
