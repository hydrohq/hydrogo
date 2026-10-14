# Reading production logs for shared hosting / CGI deployments (AI guide)

Audience: an agent debugging a live issue on a shared-hosting CGI
deployment (see `deploy-ftp-shared-hosting.md` for the deployment this
assumes). Covers getting at the logs and reading what HydroGo actually
puts in them.

## Getting the raw log file when there's no panel log viewer (or it's slow)

Most control panels expose a log viewer, but it's often easier and faster
to fetch the file directly over the same FTP connection used for
deployment:
```sh
curl -s --ftp-pasv "ftp://<host>/<webspace-root>/logs/" --user "<user>:<password>" --max-time 30
```
lists what's available — typically `error_log`, `access_log`, sometimes
separate `access_ssl_log`/`proxy_*_log`. Fetch and tail:
```sh
curl -s --ftp-pasv "ftp://<host>/<webspace-root>/logs/error_log" --user "<user>:<password>" --max-time 60 | tail -200
```
For a very large log, use curl's range request (`-r <start>-<end>`) with
byte offsets from a prior `--list-only`/listing's file size rather than
downloading the whole thing every time.

## How to read what's in there

Apache's own lines are prefixed `AH0####` (e.g. `AH01215` = "stderr from
a CGI script", `AH01276` = directory-index-related). Everything HydroGo
itself logs arrives as the **stderr content of a CGI invocation**, nested
inside an `AH01215` line:
```
[Fri Oct 02 ...] [cgid:error] [pid ...] AH01215: stderr from .../hydro.cgi: {"time":"...","level":"INFO","msg":"http_request","request_id":"...","method":"GET","path":"/...","route":"/...","module":"...","status":200,"duration_ms":6,...}
```
That embedded JSON is HydroGo's own structured log — `msg:"http_request"`
lines are per-request access logs (status code, duration, route, bytes);
other `msg` values are app-level events. `grep` for a specific `path`,
`request_id`, or `status` value to isolate what you're looking for:
```sh
... | grep '"path":"/public/news/tr/html"'
... | grep '"status":500'
```

A bare error (no JSON, just plain text after the `AH01215:` prefix) means
the binary failed before or outside its normal request-handling path —
these are the ones worth reading most carefully, they're usually a fatal
config/path/permission problem rather than an application-level request
error. Compare against the troubleshooting table in
`deploy-ftp-shared-hosting.md` — most of its entries are literal strings
to grep for here.

## Confirming whether a specific request hit the app at all (cache-hit detection)

If static disk caching is active (see
`docs/architecture/0265-fastcgi-and-persistent-html-cache.md`), a cache
hit is served directly by Apache and **never produces a log line** —
`hydro.cgi` isn't invoked. To confirm a given URL is actually being served
from cache rather than re-executing every time: note the `request_id`
from one request's log line, then make several more requests to the same
URL and confirm **no new log entries appear** for it. Silence here is the
proof, not an error to chase.

## Browsing the cache directory itself (shared hosting + static cache)

```sh
curl -s --ftp-pasv "ftp://<host>/<webspace-root>/httpdocs/static_cache/<domain>/public/<pages-or-news>/<locale>/.../" --user "<user>:<password>"
```
A `.html` file's `Last-Modified` timestamp tells you when that specific
page was last (re)cached — useful for confirming whether you're looking
at a stale pre-deploy copy or a fresh one. Force a fresh write by deleting
the file (`curl ... -Q "DELE <relative-path>"`, issued against a URL that
resolves to the file's parent directory, with the path relative to that
directory) and requesting the page again.

## What NOT to do

- Don't assume a quiet log means nothing happened — for a cached route,
  quiet is the success case (see above).
- Don't download a multi-hundred-KB log repeatedly in a tight loop while
  debugging — tail what you need, or use a byte-range request.
