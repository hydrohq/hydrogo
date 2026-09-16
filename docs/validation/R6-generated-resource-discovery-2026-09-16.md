# R6 generated resource discovery validation

## Scope

Generated resources now include a deterministic `spec.json` source document.
`hydro resource check --root` can discover every collection below `generated/`
without a database, network call, or separately supplied spec path.

## Verification

From a disposable generated project:

```powershell
hydro resource new --name article --field title:text --out .\generated
hydro resource check --root .
```

Observed output includes `resource-compatible collection=article` and the
canonical spec hash. The existing explicit form remains supported:

```powershell
hydro resource check --spec .\article.yaml --root .
```

The command rejects missing, malformed, symlinked, or incomplete generated
collections. This is a tooling/wiring preflight; runtime route, policy,
migration, and module composition remain explicit review steps.
