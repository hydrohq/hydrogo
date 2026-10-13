# Redeploying a Live Site (Human Guide)

After a site is live in production, shipping subsequent code updates follows a consistent lifecycle: **merge → compile → transfer → restart/verify → push**.

## Deployment Steps

1. **Merge changes** to `main` branch.
2. **Compile the production binary** for the target operating system (e.g., `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o hydro ./cmd/hydro`).
3. **Audit modified assets:** Identify if the release includes template changes (`themes/`), config updates (`config.yaml`), `.htaccess`, or database migrations in addition to the binary. Transferring only the binary when styles or templates have changed is a common source of release drift.
4. **Deploy files:**
   - **Shared Hosting (CGI):** Transfer the binary and modified files via FTP. Because Apache invokes the binary per request, new requests execute the updated binary immediately.
   - **VPS (systemd):** Transfer the binary via `scp`, then restart the service (`sudo systemctl restart hydro`). Without restarting the systemd service, the previous process remains active in memory.
5. **Execute database migrations** if schema changes are included (`hydro migrate`).
6. **Verify production:** Open `https://your-domain.com/` and the Admin Studio. Confirm the application renders without errors.
7. **Push to GitHub:** Push the reviewed commits to GitHub to keep git history synchronized with production.
