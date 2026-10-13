# WSL local baseline — 2026-09-16

**Environment:** Windows WSL2, Ubuntu, `Linux DESKTOP-RL4NSEN 6.18.33.2-microsoft-standard-WSL2`, `x86_64`  
**User:** `testuser`  
**Go:** `go1.27.0 linux/amd64`

## Command

```bash
cd /path/to/project
go test ./...
```

## Result

The test suite completed successfully. No test failures were observed in `cmd/hydro`, extensions, internal packages, modules, migrations, contract tests, or tools packages. Some packages ran from cache following source download; this report serves as a WSL local baseline evidence rather than full production certification.

## Scope & Boundaries

This execution does not substitute for:

- Restore and capacity verification on a clean PostgreSQL target
- Live reverse-proxy blue/green switchover
- External S3/CDN, SMTP, and backup provider drills
- Cross-platform signed release
- Production RPO/RTO and customer hardware measurement

No passwords were written to this document or command output.
