# WSL full suite — post permission contract — 2026-09-16

- Ubuntu WSL2, user `testuser`
- Directory: `/path/to/project`
- Command: `go test ./... -p 1 -timeout=5m`
- Result: exit code 0; all packages returned `ok` or `[no test files]`.

This run covers permission preflight, generator verification, and all prior SDK/release changes. External production services remain outside the scope of this test.
