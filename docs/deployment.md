# Deployment

HydroGo is one static binary plus a config file and a data directory. Pick one of these.

## VPS or any Linux server

1. Install the binary (`install.sh`, see [getting started](getting-started.md)) and create the project on the server with `hydro new`, or copy yours.
2. Set the production values in `config.yaml` (or `HYDRO_*` environment variables):
   `http.public_origin: "https://cms.example.com"`, `security.secure_cookies: true`, `security.enable_hsts: true`. Enroll admins in MFA before using `mfa_required_roles`.
3. Run it under systemd:

```ini
# /etc/systemd/system/hydrogo.service
[Service]
User=hydro
WorkingDirectory=/srv/mysite
ExecStart=/usr/local/bin/hydro serve --config config.yaml
Restart=on-failure
```

4. Put Caddy or nginx in front for TLS: see [`deploy/caddy`](../deploy/caddy) and [`deploy/nginx`](../deploy/nginx).

## Docker

`ghcr.io/ardaozyildirim/hydrogo` (linux/amd64 and arm64). Mount a project at `/app` or a config at `/app/config.yaml`, and a volume at `/app/data`. [`deploy/docker-compose.prod.yml`](../deploy/docker-compose.prod.yml) is a full example with Caddy.

Building the image yourself needs BuildKit (`docker buildx`, included in current Docker releases); the classic builder cannot read `--platform=$BUILDPLATFORM` and stops at the first stage. Build from a clean `git clone`: `.dockerignore` keeps `node_modules` and `.svelte-kit` out of the context, but a checkout with its own `admin/node_modules` is the usual cause of a failing `npm run build` stage.

Order matters on a new database: `hydro migrate`, then `hydro create-admin` (and, for staff who are not administrators, `hydro create-user --email ... --role viewer --permissions a,b --password-stdin`), then `hydro serve`. With Docker that is `docker compose run --rm -T hydrogo migrate --config /app/config.yaml` before `up`. `serve` refuses an unmigrated database and says so.

Set `HYDRO_PUBLIC_ORIGIN` (or `http.public_origin`) to the address people use. HydroGo answers `421` to any other `Host` header, and the message names the host it expects; a service in the same Docker network that calls `http://hydrogo:8080` must send the public host in its `Host` header.

## Invitations and password resets

An invitation (`POST /api/v1/admin/users/invites`, needs SMTP) and a password reset (`POST /api/v1/admin/users/{id}/password-reset-link`, or the e-mail a person requests) end in a link to the admin panel, `/admin/accept-invite?token=...` and `/admin/reset-password?token=...`. Those pages set the password and send the person to `/admin/login`. Without SMTP, an administrator can still copy the reset link from the Users screen and pass it on, and `hydro create-user` opens accounts directly.

## Shared hosting (CGI/FastCGI)

See [`deploy/hosting/README.md`](../deploy/hosting/README.md). Use PostgreSQL or MySQL once traffic grows beyond a small site.

## Databases

SQLite (default) for small sites; PostgreSQL or MySQL/MariaDB for more traffic or several instances (`database.dialect`, `HYDRO_DATABASE_DSN`). Add Redis (`HYDRO_REDIS_URL`) for shared rate limits and caches when you run more than one instance.

## Backups

`hydro backup sqlite --destination backups/ --media-dir data/media` (or `postgres`), and test `hydro restore` on a scratch copy before you need it. Run it from cron or a systemd timer.

On PostgreSQL the audit tables use forced row-level security, so the application's own database user cannot dump or restore them. Set `HYDRO_PG_PRIVILEGED_ROLE` to a role that user may `SET ROLE` to: for backups, one with `BYPASSRLS` and `pg_read_all_data`; for `hydro restore postgres` into an empty database, one that can also create objects there. Without it the command stops with a hint instead of writing an incomplete backup.

## Frontend

Deploy your frontend separately and point it at the API. The SvelteKit starter runs as a Node server (`npm run build && npm start`) next to HydroGo.

## Tamper-evident audit log

Audit events are already immutable in the database (triggers reject updates
and deletes). For businesses that must show the log was not altered, turn on
the audit chain:

```yaml
audit:
  chain: true          # on by default in projects made with --preset erp-lite
```

Every new audit event then gets a link: its position per tenant and a SHA-256
over the previous link and the event's fields. Writers queue on the tenant's
head row (one extra row lock per audit write: audited writes of a tenant are serialized), so the chain never forks.
Events written before the chain was switched on stay outside it.

```bash
hydro audit verify --config config.yaml    # recomputes every link; exit 1 if anything is off
hydro audit head   --config config.yaml    # JSON: the latest position and hash per tenant
hydro audit relink --config config.yaml   # link events written while audit.chain was off
hydro audit verify --anchor :1520:9f2c…    # TENANT:SEQ:HASH, a head you saved earlier
```

`verify` reports a changed event, a missing event or link, a moved head and
events that were written after the chain began but are not in it. Someone
who can write to the database (and lifts the triggers) could rebuild the
entire chain consistently, so save the output of `hydro audit head` outside
the database now and then (a ticket, a mailbox, WORM storage) and pass it as
`--anchor`: verification then proves that the exact link you saved is still
in the chain. MySQL is covered by statement tests only; PostgreSQL and
SQLite are exercised with concurrent writers.

## Backing up MySQL

`hydro backup mysql` runs the client tools of the same family as the server: MySQL's `mysqldump` and `mysql` for a MySQL server. MariaDB's tools write the values of generated columns, which a MySQL server refuses on restore, so a backup made that way could not be restored; HydroGo stops before writing it and says so. Install the MySQL client, or point `HYDRO_MYSQLDUMP` and `HYDRO_MYSQL` at it. MariaDB tools against a MariaDB server are fine.

