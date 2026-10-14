# WSL full suite evidence — 2026-09-16

## Environment

- Distribution: Ubuntu (WSL2)
- User: `testuser`
- Working tree: `/path/to/project`
- Command: `go test ./... -p 1 -timeout=5m`

## Result

The test process completed successfully (exit code 0). All Go packages returned `ok` or `[no test files]`; `internal/sdkdist`, `internal/projectgen`, `internal/release`, and `cmd/hydro` were included.

This evidence demonstrates local WSL behavior. It is not verification of PostgreSQL, real artifact broker, signing keys, S3/CDN, or production installers.
