# R5 — Permission contract check

`hydro permissions check --spec SPEC [--json]` now verifies permission names in resource specs offline. Whitespace, traversal-like names, uppercase letters, and duplicate permissions are rejected; valid specs report only collection and permission counts.

Windows:

```text
go test ./internal/generator ./cmd/hydro
PASS
```

WSL Ubuntu (`testuser`):

```text
go test ./internal/generator ./cmd/hydro ./internal/release ./internal/sdkdist
exit code 0
```
