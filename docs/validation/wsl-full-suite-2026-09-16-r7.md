# WSL full suite — post validated broker grant — 2026-09-16

- Ubuntu WSL2, user `testuser`
- Command: `go test ./... -p 1 -timeout=5m`
- Result: exit code 0; all packages returned `ok` or `[no test files]`.

This run covers `RequestValidatedGrant`, broker response boundaries, artifact hygiene, and all previous generator/CLI/runtime phases. Real OAuth/broker accounts and production key ceremonies remain outside the scope of this report.
