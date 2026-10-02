# Shared hosting

## CGI

Build the single binary with `make build` (or cross-compile: `GOOS=linux
GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=false
-ldflags="-s -w -buildid=" -o hydro ./cmd/hydro`). CGI mode requires the
explicit `serve --cgi` subcommand; a plain Apache CGI invocation passes no
argv, only environment variables, so the document root's `hydro.cgi` cannot
be the raw binary. Keep the binary and `config.yaml` outside the document
root (a sibling `private/` directory, not `httpdocs/`) so they are never
web-servable, and make `hydro.cgi` a wrapper script that execs the real
binary with explicit flags:

```sh
#!/bin/sh
exec /path/to/private/hydro serve --cgi --config /path/to/private/config.yaml
```

Upload this wrapper (and any other `.cgi` file) as a binary/text file via
SFTP or the control panel's file **upload**, not by pasting into an in-panel
"new file" text editor — panel-created files can get different
owner/group than uploaded files, which trips `suexec`'s ownership check and
fails every request with `fopen: Permission denied` logged against
`suexec_log`, never reaching the application.

`.htaccess` needs a `DirectoryIndex hydro.cgi` line. Without it, a request
for `/` has no matching index file once any placeholder `index.html` is
removed from the document root; with directory listing disabled
(`Options -Indexes`), Apache returns `403 Forbidden` before the CGI
handler ever runs.

Apache then passes existing files and directories through directly and
sends other paths to HydroGo. A CGI request starts a fresh process, so CGI
mode avoids resident workers. Scheduled News jobs run in bounded batches
during public requests.

### SQLite vs MySQL under CGI

CGI's process-per-request model means concurrent requests open concurrent
database connections with no shared pool. SQLite's single-writer file lock
makes this risky under any real concurrency; prefer the MySQL or PostgreSQL
adapter for CGI hosting when the provider offers one, reserving SQLite for
low-traffic or single-admin deployments.

### Running one-shot commands without shell access

Many shared-hosting plans with CGI support provide neither SSH nor a
scheduled-task runner, so `hydro migrate` and `hydro create-admin` have no
obvious execution path. A `.cgi` script can stand in for a shell: give it a
long random token checked against `QUERY_STRING` before running anything,
return `404` on any mismatch, run the target command(s) with their output
echoed back, and delete the script immediately after use — it holds no
state and is not meant to persist.

```sh
#!/bin/sh
TOKEN="<random-hex>"
case "$QUERY_STRING" in
  token=$TOKEN)
    echo "Content-Type: text/plain"; echo ""
    /path/to/private/hydro migrate --config /path/to/private/config.yaml 2>&1
    ;;
  *)
    echo "Status: 404 Not Found"; echo "Content-Type: text/plain"; echo ""; echo "not found"
    ;;
esac
```

CGI mode writes eligible anonymous public page and news HTML to
`<document-root>/static_cache` so Apache can serve a cache hit without starting
CGI. Only known public HTML routes are stored; session/authenticated requests,
HTMX requests, query variants, responses that set cookies or contain a CSP
nonce, errors, and oversized documents bypass this cache. The cache is capped
at 128 entries and 32 MiB. Page and News changes clear it. Clear the directory
after making out-of-band database changes. The `.htaccess` denies direct
requests to the cache directory and applies a fixed CSP to HTML responses.

Eligible public page and news-article HTML routes no longer need the CSRF
cookie to exist before the response can be cached: their only CSRF-dependent
widget (the "was this helpful" feedback control) fetches its token
client-side from `GET /csrf-token` instead of having it baked into the
shared HTML. See `internal/security/csrf.go`'s `skipCookieFor` parameter and
`internal/server/server.go`'s `publicContentCacheEligible`.

Every cache write also produces a pre-compressed `<file>.html.br` sibling
(`internal/server/static_cache.go`'s `writeCachedFileBrotliSibling`, reusing
the same Brotli encoder as the dynamic-response `CompressionMiddleware` in
`internal/server/http_optimization.go`). A disk-cache hit is served by Apache
alone and never reaches that middleware, so without the pre-compressed
sibling a cached response would always go out uncompressed regardless of
what the live CGI path would have sent. `.htaccess` prefers the `.br` file
whenever the client's `Accept-Encoding` includes `br`, falling back to the
plain `.html` file otherwise; compression failures are non-fatal and just
leave that one response uncompressed. Responses are also eligible for
LiteSpeed's own edge cache: eligible responses carry an
`X-LiteSpeed-Cache-Control: public,max-age=30` header (same TTL as
`internal/capacity`'s `CacheTTL`), which is a no-op on plain Apache and an
additional cache layer on LiteSpeed Web Server (a common Apache-compatible
drop-in on shared hosting) without any separate configuration.

## FastCGI

FastCGI is a long-running responder. Start it under the hosting provider's
process manager with:

```sh
hydro serve --fastcgi --fastcgi-address 127.0.0.1:9001
```

Configure the provider's Apache FastCGI integration to connect to that
loopback address. The port is intentionally not bound to public interfaces by
default. `hydro serve` retains its regular bounded background workers in this
mode and drains requests on shutdown.
