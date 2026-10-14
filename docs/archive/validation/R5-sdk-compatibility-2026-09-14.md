# R5 SDK compatibility validation — 2026-09-14

- `go test ./extension/...` passes.
- `extension/v1/compatibility_test.go` asserts the public module and theme
  contracts at compile time, including the optional BlockTheme extension.
- Experimental v0 module/theme contracts remain source-compatible and expose
  Go deprecation guidance toward the versioned SDK.
