# WSL full suite — post schema diff — 2026-09-16

- Distribution: Ubuntu WSL2
- User: `testuser`
- Directory: `/path/to/project`
- Command: `go test ./... -p 1 -timeout=5m`
- Result: exit code 0; all packages returned `ok` or `[no test files]`.

This run covers signed schema compatibility diff and associated release tests. External services and production deployment gates must be validated separately.
