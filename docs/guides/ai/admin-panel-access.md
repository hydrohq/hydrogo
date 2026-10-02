# Admin Studio panel access (AI guide)

Audience: an agent that needs to create an admin account and log into the
Admin Studio panel, either on a local dev checkout or a deployed instance.

## Key fact: no admin account exists by default

Neither `hydro run` (local dev) nor a fresh production deploy creates an
admin user automatically. `hydro seed-demo` deliberately creates only a
demo **member** account (`internal/auth` audience, not admin) — see the
comment at its call site in `cmd/hydro/main.go`: "Demo member data is
deliberately isolated from admin users... gives the obvious demo
credential no administrator capability." You must always run
`create-admin` yourself.

## Create the admin account

```sh
hydro create-admin --email <email> --password-stdin --config <config.yaml>
```
The password is read from **stdin**, not a flag — it never appears in shell
history or a process list. Pipe it in:
```sh
printf '%s\n' "<password>" | hydro create-admin --email <email> --password-stdin --config <config.yaml>
```
Success output: `Created admin user: <email> (id: <uuid>)`.

If you don't have shell access to the target (shared hosting, no SSH), use
the one-shot token-gated CGI script pattern from
`docs/guides/ai/deploy-ftp-shared-hosting.md` section 6 — same command,
wrapped so it can be triggered once over HTTPS and then deleted.

## Log in

- Local dev (`hydro run`): `http://127.0.0.1:8090/login`
- Deployed: `https://<domain>/login`
- `/admin` redirects to `/login` if not authenticated (HTTP 303), so either
  URL works as an entry point.

Login is a standard email+password POST to `/auth/login`. A `401` response
means wrong credentials, not a server problem — don't assume a bug before
checking the email/password are exactly right (case-sensitive email
normalization and password are both exact-match).

## Roles and permissions

Three roles exist: `admin`, `editor`, `viewer`
(`internal/auth`, `auth.Role*` constants). `create-admin` always creates an
`admin`-role user. To change a user's role after creation, use Admin
Studio's user management screen (`/admin/...`) as an admin, or the
`UpdateRole` repository method if scripting it — there is no standalone
CLI subcommand for role changes as of this writing (check
`cmd/hydro/main.go`'s top-level `switch` for new subcommands before
assuming this is still true).

## MFA

Admin Studio supports TOTP-based MFA (`internal/auth/sqlstore/mfa_store.go`,
recovery codes included). It's opt-in per user via the panel, not enforced
by default unless `security.mfa_required_roles` in `config.yaml` lists the
role. If you lock yourself out of a local dev admin account during testing,
the fastest recovery is usually re-running `create-admin` with a new email
(or dropping the dev SQLite file and starting over — it's disposable demo
data, not production).

## Verifying access programmatically

```sh
curl -s -c /tmp/cookies.txt -b /tmp/cookies.txt "http://127.0.0.1:8090/login" -o /dev/null -w "%{http_code}\n"
```
then POST credentials with the CSRF token extracted from that page's
`hydro_csrf` cookie and the form's `_csrf` hidden field (see
`internal/security/csrf.go` — it's a double-submit cookie scheme, cookie
value must match the form field exactly). For most agent tasks it's
simpler to just confirm `GET /admin` returns `200` after a successful
browser-based login than to script the full login flow.
