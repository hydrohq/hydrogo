# Deploying to Shared Hosting via FTP (Human Guide)

This guide covers how to deploy HydroGo in CGI mode to a standard shared hosting account (cPanel, Plesk, DirectAdmin, etc.) **without SSH access or scheduled cron tasks**, from initial setup to production verification. For automated or agent workflows, refer to the companion guide [`docs/guides/ai/deploy-ftp-shared-hosting.md`](../ai/deploy-ftp-shared-hosting.md).

## Why FTP and CGI?

HydroGo is normally operated as a persistent standalone daemon. On shared hosting without SSH or daemon supervisor privileges, Apache/LiteSpeed manages incoming requests by invoking the binary in **CGI mode** (`hydro serve --cgi`). **FTP/SFTP** is the universal protocol supported across all shared hosts to upload assets and binaries.

## Prerequisites

- **FTP credentials:** Hostname (domain or server IP), username, and password.
- **Database details:** Database name, username, and password created in your hosting panel (MySQL / MariaDB recommended; SQLite file-locking can contend with CGI process models under concurrency).
- **CGI support enabled:** Ensure CGI scripts (`.cgi`) are enabled in your hosting panel settings.

## 1. Verify FTP Connectivity

Connect via an FTP client like FileZilla or Cyberduck. Upon connection, observe your webspace root directory (typically containing `public_html` or `httpdocs`, `logs`, etc.).

## 2. Directory Structure

Isolate private binaries and credentials from publicly accessible document roots:

```text
<webspace-root>/
  private/            ← NOT accessible via web; holds binary and secrets
    hydro
    config.yaml
    themes/<theme>/
    media/
  httpdocs/            ← Public web root (or public_html/)
    hydro.cgi
    .htaccess
```

Keep `private/` outside of `httpdocs/` so that `config.yaml` and database credentials can never be downloaded directly via HTTP.

## 3. Compile the Binary

Cross-compile the binary for the host architecture (typically Linux amd64):

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o hydro ./cmd/hydro
```

## 4. Upload Files via FTP

Upload `hydro` to `private/`, and upload `hydro.cgi` and `.htaccess` to `httpdocs/`. Ensure `hydro` and `hydro.cgi` have **executable permissions (755)** (right-click file in FTP client → Permissions).

**Important Note on suexec:** Do not create `hydro.cgi` by copy-pasting into web panel file editors. Web editors can assign differing file ownership, triggering suexec permission errors (`fopen: Permission denied`). Always prepare the file locally and upload it directly.

`hydro.cgi` wrapper script:
```sh
#!/bin/sh
exec /absolute/path/to/private/hydro serve --cgi --config /absolute/path/to/private/config.yaml
```

## 5. Prepare `config.yaml`

Configure database connection strings, theme names, and public origin. Ensure YAML indentation uses clean spaces without tabs.

## 6. Run Migrations and Bootstrap Admin

Without SSH, migrations cannot be triggered directly via shell. Use a single-use token-gated CGI script pattern or pre-seed the database:
- Trigger the migration through an authenticated one-shot endpoint or script.
- Create the initial admin user.
- **Delete the one-shot bootstrap script immediately after use** to prevent security exposure.

## 7. Verification

Open `https://your-domain.com/` in your browser. Verify the home page renders correctly. Navigate to `/login`, sign into the Admin Studio, and confirm dashboard operations function without errors.

## 8. Troubleshooting via Logs

Check your hosting error logs (`logs/error_log` or panel Log Viewer). HydroGo errors are surfaced under `stderr from .../hydro.cgi:`.

## Common Issues & Solutions

| Symptom | Cause | Resolution |
|---|---|---|
| Home page returns 403 Forbidden | Missing `DirectoryIndex hydro.cgi` in `.htaccess` or leftover `index.html` | Update `.htaccess` and remove placeholder `index.html` |
| `fopen: Permission denied` | Script created in web editor with bad ownership | Upload file cleanly via FTP |
| "No such file or directory" | Incorrect absolute path in `hydro.cgi` | Verify full path to binary |
| `decode config: yaml: ...` | YAML syntax/indentation error | Validate `config.yaml` formatting |
| `Table '...' doesn't exist` | Migrations have not run | Run migrations via bootstrap script |
