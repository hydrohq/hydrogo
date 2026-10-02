# Deploy to CGI shared hosting via FTP (AI guide)

Audience: an agent with Bash/shell and `curl` available, given FTP
credentials and a target domain. Produces a live, migrated HydroGo
deployment on Apache/LiteSpeed shared hosting (Plesk-style) using CGI mode.
No SSH or scheduled-task access is assumed — if the host has SSH, prefer it
for the one-shot commands in step 5 and skip the token-script workaround.

Background reading if something here doesn't make sense: `deploy/hosting/README.md`
(repo), `docs/architecture/0264-cgi-hosting-runtime.md`,
`docs/architecture/0265-fastcgi-and-persistent-html-cache.md`.

## Inputs you need before starting

- FTP host (try the domain itself first, fall back to the server's raw IP),
  username, password.
- The webspace root's real filesystem path (e.g.
  `/var/www/vhosts/<domain>/`) — confirm it, don't assume; get it from an
  `EPSV`/`PWD` listing or ask the operator.
- A database the app can reach: MySQL/PostgreSQL strongly preferred over
  SQLite for CGI (process-per-request has no connection pool; SQLite's
  single-writer lock is a real risk under any concurrency). Credentials:
  host (usually `localhost`), db name, user, password.
- Confirm the hosting plan has CGI enabled (not just PHP/FastCGI — those are
  separate toggles in most panels).

## 1. Test the FTP connection (read-only)

```sh
curl -s --list-only "ftp://<host>/" --user "<user>:<password>" --max-time 20
```

If this times out against the domain, retry against the raw server IP with
`--ftp-pasv` explicit and `-v` to see the negotiation:

```sh
curl -s --ftp-pasv "ftp://<ip>/" --user "<user>:<password>" --max-time 20 -v
```

A working login shows `230 User ... logged in` and a directory listing
(typically `httpdocs`, `logs`, `private` or similar). That listing also
tells you the real webspace-root layout — use it, don't guess.

## 2. Lay out the remote directory structure

```
<webspace-root>/
  private/            # NOT web-servable — binary, config, secrets live here
    hydro
    config.yaml
    themes/<theme>/
    media/
  httpdocs/            # the actual document root
    hydro.cgi
    .htaccess
```

`private/` must be a **sibling** of `httpdocs/`, never inside it — anything
under `httpdocs/` is web-fetchable, and `config.yaml` holds the DB password.

Create `private/` via FTP `MKD` if it doesn't exist:
```sh
curl -s --ftp-pasv "ftp://<host>/<webspace-root>/" --user "<user>:<password>" -Q "MKD private"
```

## 3. Build the binary

From the repo root, cross-compile (works from any OS):

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o dist/hydro ./cmd/hydro
```

(Or `make build-hosting` / `scripts/build-hosting.ps1` if this repo has
those targets — check `Makefile`/`scripts/` first.) Both the SQLite and
MySQL drivers are pure Go; no C toolchain is needed on either end.

## 4. Upload the binary and support files via FTP

**Binary mode is automatic for `-T` uploads in curl — don't second-guess
it.** Upload `hydro` to `private/`:

```sh
curl -s --ftp-pasv -T "dist/hydro" "ftp://<host>/<webspace-root>/private/hydro" --user "<user>:<password>" --max-time 180 -w "uploaded: %{size_upload} bytes\n"
```

Verify the uploaded byte count matches the local file size exactly.

Check permissions after upload:
```sh
curl -s --ftp-pasv "ftp://<host>/<webspace-root>/private/" --user "<user>:<password>"
```
If this is the **first** upload of this file, it likely lands as `644` (no
execute bit). Try `SITE CHMOD`:
```sh
curl -s --ftp-pasv -T "dist/hydro" "ftp://<host>/<webspace-root>/private/hydro" --user "<user>:<password>" --quote "-SITE CHMOD 755 private/hydro"
```
Some FTP daemons (ProFTPD with `SITE CHMOD` disabled) reject this with `550`
— that's fine to ignore (`--quote` prefixed with `-` is meant to not fail
the transfer) as long as the file already shows `rwxr-xr-x` in the listing.
**On re-uploads that replace an already-`755` file, most FTP servers
preserve the existing permission bits** — re-check the listing before
assuming you need to chmod again.

Upload `.htaccess` and the `hydro.cgi` wrapper (content below) to
`httpdocs/` the same way. **Never create `.cgi` files through a hosting
panel's in-browser text editor** — panel-created files commonly get
different ownership/group than FTP-uploaded ones, which trips `suexec`'s
ownership check and produces this exact failure in the error log with zero
useful detail otherwise:
```
AH01215: stderr from .../hydro.cgi: failed to open log file /var/log/httpd/suexec_log
AH01215: stderr from .../hydro.cgi: fopen: Permission denied
```
If you see that, re-upload the file via FTP instead of fixing permissions.

`hydro.cgi` (`httpdocs/hydro.cgi`, chmod 755):
```sh
#!/bin/sh
exec <webspace-root>/private/hydro serve --cgi --config <webspace-root>/private/config.yaml
```
Apache passes **no argv** to a CGI script, only environment variables —
this wrapper exists because `hydro`'s CGI mode requires the explicit
`serve --cgi` flag; `hydro.cgi` cannot be the raw binary.

`.htaccess` — copy `deploy/hosting/.htaccess` from the repo as-is. It
already includes `DirectoryIndex hydro.cgi` (required — see troubleshooting
below if an older copy is missing it) plus the static-cache passthrough
rules.

## 5. Config file

`private/config.yaml` — adapt from `deploy/hosting/README.md`'s example.
Minimum for a working deploy:
```yaml
profile: compact
modules: [core, auth, members, pages, media, site, news]
http:
  address: "127.0.0.1:8080"
  public_origin: "https://<domain>"
database:
  dialect: "mysql"
  dsn: "<db_user>:<db_password>@tcp(localhost:3306)/<db_name>"
media:
  dir: "<webspace-root>/private/media"
theme_dir: "<webspace-root>/private/themes"
theme: "<theme>"
security:
  secure_cookies: true
  enable_hsts: true
  headers_preset: strict
  enable_anomaly_filter: true
```
YAML is whitespace-sensitive — a stray leading space on a top-level key
(e.g. `" mail:"` instead of `"mail:"`) breaks the whole file with
`hydro: decode config: yaml: line N: did not find expected key` and takes
the entire site down with it. If you hit that, diff the file's indentation
line by line rather than guessing.

## 6. Run migrations and create the admin user (no SSH)

If SSH exists, just run `hydro migrate --config ...` and
`hydro create-admin --email ... --password-stdin --config ...` directly and
skip to step 7.

Without SSH, build a **temporary, token-gated CGI script**, upload it via
FTP (same ownership rule as step 4 applies), hit it once over HTTPS, then
delete it immediately — it holds secrets in plain text and must not persist.

`setup.cgi` (generate a random hex token first, e.g. `openssl rand -hex 16`):
```sh
#!/bin/sh
TOKEN="<random-hex>"
BIN="<webspace-root>/private/hydro"
CFG="<webspace-root>/private/config.yaml"
case "$QUERY_STRING" in
  token=$TOKEN)
    echo "Content-Type: text/plain"; echo ""
    "$BIN" validate --config "$CFG" 2>&1
    "$BIN" migrate --config "$CFG" 2>&1
    "$BIN" doctor --config "$CFG" 2>&1
    ;;
  *)
    echo "Status: 404 Not Found"; echo "Content-Type: text/plain"; echo ""; echo "not found"
    ;;
esac
```
Visit `https://<domain>/setup.cgi?token=<random-hex>`, confirm no errors in
the output, then delete the file via FTP.

Admin creation follows the identical pattern, piping the password into
stdin instead of a flag (`create-admin` reads the password from stdin, not
argv, so it's never visible in a request log or process list):
```sh
#!/bin/sh
TOKEN="<random-hex>"
BIN="<webspace-root>/private/hydro"
CFG="<webspace-root>/private/config.yaml"
EMAIL="<admin-email>"
PASSWORD="<generated-password>"
case "$QUERY_STRING" in
  token=$TOKEN)
    echo "Content-Type: text/plain"; echo ""
    printf '%s\n' "$PASSWORD" | "$BIN" create-admin --email "$EMAIL" --password-stdin --config "$CFG" 2>&1
    ;;
  *)
    echo "Status: 404 Not Found"; echo "Content-Type: text/plain"; echo ""; echo "not found"
    ;;
esac
```
Visit once, confirm `Created admin user: ...`, delete the file immediately.
Tell the operator the generated password out-of-band (chat is fine for a
private session) and recommend they change it after first login.

## 7. Verify

```sh
curl -s -D - -o /dev/null "https://<domain>/" --max-time 15
```
Expect `HTTP/1.1 200`. If you get `500`, read the error log (step 8). If
you get `403` before any `hydro.cgi` stderr appears in the log, see the
`DirectoryIndex` entry below.

Log in at `https://<domain>/login` with the admin credentials from step 6,
confirm the admin panel loads and basic navigation (e.g. listing
pages/media) works without a 500.

## 8. Reading the error log over FTP

Panels that don't expose a log viewer still let you fetch the raw file:
```sh
curl -s --ftp-pasv "ftp://<host>/<webspace-root>/logs/" --user "<user>:<password>" --max-time 30
curl -s --ftp-pasv "ftp://<host>/<webspace-root>/logs/error_log" --user "<user>:<password>" --max-time 60 | tail -100
```
`hydro`'s own structured JSON logs appear inline inside Apache's
`stderr from .../hydro.cgi:` lines — grep for `http_request` to see
per-request app-level logs, or for the specific path you're debugging.

## Troubleshooting matrix (all observed on a real first deployment)

| Symptom | Cause | Fix |
|---|---|---|
| `403 Forbidden`, no `hydro.cgi` log entry at all | `DirectoryIndex hydro.cgi` missing from `.htaccess`, or an `index.html` placeholder still exists in `httpdocs/` shadowing it | Add the directive; delete any placeholder `index.html` |
| `fopen: Permission denied` / `failed to open log file ... suexec_log` | File was created via the hosting panel's text editor, not uploaded | Delete, re-upload via FTP |
| `hydro.cgi: line 2: <path>: No such file or directory` | The hardcoded path in the wrapper script doesn't match the real webspace root | Re-verify the root via an FTP listing, fix the path, re-upload |
| `Undeclared variable: $2` (or similar `$N`) with MySQL | A hand-written SQL query defaulted to Postgres placeholder syntax with no MySQL branch — a real upstream bug class, not a config issue | Check if already fixed in current `main`; if not, this needs a code fix, not a config change |
| `Table '...' doesn't exist` | Migrations were never run | Run step 6 |
| `DNS_PROBE_FINISHED_NXDOMAIN` right after a DNS change | Nameserver propagation in progress | Not fixable from here; wait, suggest `dnschecker.org`, a DNS flush, or testing from a different network |
| `hydro: decode config: yaml: line N: did not find expected key` | Bad YAML indentation in `config.yaml` | Diff indentation line by line, most likely a stray leading space on a top-level key |
| `gofmt required: <file>` on `git push`, file shows as modified with no real diff (`git diff -b` empty) | Windows CRLF line endings reintroduced into a file `.gitattributes` marks `eol=lf`, usually left behind by an editor or script that wrote the file in text mode | `gofmt -w <file>` to renormalize, then re-push |

## What NOT to do

- Don't use `hydro serve --fastcgi` unless you can keep a persistent process
  alive (needs SSH + a process manager). Shared hosting without SSH only
  supports plain CGI (`serve --cgi`).
- Don't leave any token-gated `.cgi` script (step 6) on the server after
  use — delete it the same session you create it.
- Don't put `config.yaml`, the binary, or `media/` inside `httpdocs/`.
- Don't skip verifying the uploaded binary's byte count matches the local
  file — a truncated FTP transfer fails silently otherwise.
