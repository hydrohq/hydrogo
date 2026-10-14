# Local bootstrap

HydroGo local development can run entirely inside WSL Ubuntu or with the
optional `compose.dev.yaml` dependency profile. These settings are development
only; the compose file binds services to localhost and uses non-production
credentials.

## WSL

From the repository root inside WSL:

```bash
bash scripts/bootstrap-wsl.sh
go test ./... -p 1
```

The script checks Go, PostgreSQL client/server readiness and Redis. It does not
alter application configuration or create production data.

## Docker Compose

```bash
docker compose -f compose.dev.yaml up -d
docker compose -f compose.dev.yaml ps
docker compose -f compose.dev.yaml down
```

Use a local-only DSN such as
`postgres://hydrogo:hydrogo-dev-only@127.0.0.1:5432/hydrogo?sslmode=disable`.
Never reuse these credentials outside a development machine.
