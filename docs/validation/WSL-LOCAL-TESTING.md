# WSL local infrastructure and security testing

Run the application from WSL with `scripts/bootstrap-wsl.sh`, then execute:

```bash
go test ./...
go test -race ./...
go vet ./...
```

Optional services:

- PostgreSQL: run the certified DB contract suite with a disposable database.
- Redis/Valkey: set `HYDRO_REDIS_URL` and test distributed rate limits, TTL and
  connection loss.
- MinIO/LocalStack: configure `HYDRO_MEDIA_STORAGE_DRIVER=s3` with a test
  endpoint/bucket and exercise upload, read, delete, metadata and range flows.

Security checks must target only an owned local instance and synthetic data:

1. OWASP ZAP baseline and authenticated active scan.
2. `ffuf` endpoint/parameter discovery and `nmap` local port inventory.
3. `testssl.sh` against the local TLS reverse proxy when enabled.
4. Manual authorization matrix: IDOR, privilege escalation, CSRF, session
   fixation, upload bypass, path traversal, SSRF and rate-limit bypass.
5. Go fuzz and race suites for parsers, redirects, uploads and webhooks.

Local scans are engineering verification, not an independent production
penetration test. Production testing requires written authorization, scope,
staging data and a rollback window.
