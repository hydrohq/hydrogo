# Reading Production Logs (Human Guide)

When troubleshooting runtime anomalies (such as HTTP 500 errors or rejected requests) on a live server, the root cause is captured in the **server error log**.

## Locating Logs

- **Plesk / cPanel:** Open the "Logs" or "Log Viewer" tool in your hosting control panel, or download `logs/error_log` via FTP.
- **VPS (systemd / Nginx):** Inspect with `journalctl -u hydro -n 100 --no-pager` or check `/var/log/nginx/error.log`.

## Interpreting HydroGo Log Entries

### Standalone Daemon (systemd / Docker)
HydroGo outputs structured JSON logs to standard error. Each entry includes:
- `timestamp`: Event time (UTC)
- `level`: `INFO`, `WARN`, `ERROR`
- `path`: Request URI
- `status`: HTTP response status code
- `duration_ms`: Execution time in milliseconds
- `error`: Formatted error message (with sensitive parameters stripped)

### CGI Mode (Apache / LiteSpeed)
In CGI shared hosting environments, HydroGo output is captured inside Apache's error logs, prefixed with:
```text
AH01215: stderr from .../hydro.cgi: {"timestamp":...,"status":500,"error":"..."}
```

If an error line following `AH01215:` contains plain text rather than JSON (e.g. `No such file or directory` or `exec format error`), this indicates an OS-level startup failure prior to Go runtime initialization (such as bad architecture compilation or invalid script paths).

## Why Cache Hits Produce No Log Entries

When disk caching or static proxy caching is active, cached responses are delivered directly by Apache/Nginx. In this state, the HydroGo binary is never invoked, meaning 0ms latency and no execution log line is generated. Seeing no log entry for a cached page view is expected and confirms caching is functioning properly.
