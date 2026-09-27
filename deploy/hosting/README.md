# Shared hosting

## CGI

Build the single binary with `make build`, copy it to the document root as
`hydro.cgi`, make it executable, and place `.htaccess` beside it. Apache then
passes existing files and directories through directly and sends other paths
to HydroGo. A CGI request starts a fresh process, so CGI mode avoids resident
workers. Scheduled News jobs run in bounded batches during public requests.

CGI mode writes eligible anonymous public page and news HTML to
`<document-root>/static_cache` so Apache can serve a cache hit without starting
CGI. Only known public HTML routes are stored; session/authenticated requests,
HTMX requests, query variants, responses that set cookies or contain a CSP
nonce, errors, and oversized documents bypass this cache. The cache is capped
at 128 entries and 32 MiB. Page and News changes clear it. Clear the directory
after making out-of-band database changes. The `.htaccess` denies direct
requests to the cache directory and applies a fixed CSP to HTML responses.

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
