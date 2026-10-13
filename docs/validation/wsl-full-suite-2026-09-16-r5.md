# WSL full suite — post artifact hygiene — 2026-09-16

- Ubuntu WSL2, user `testuser`
- Command: `go test ./... -p 1 -timeout=5m`
- Result: exit code 0; all packages returned `ok` or `[no test files]`.

This run covers release manifest artifact hygiene and SDK runtime distribution changes. External signing keys, brokers, S3/CDN, and production topologies remain outside the scope of this local suite.
