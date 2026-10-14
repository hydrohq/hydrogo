# R5 demo seed boundary validation — 2026-09-14

- `go test ./cmd/hydro -run TestSeedDemoRequiresExplicitConfirmation -count=1`
  passes.
- E2E seed invocation now supplies `--confirm-demo` explicitly.
- Production image guidance excludes source/examples and demo data from the
  final stage; demo injection is a deliberate local command.
