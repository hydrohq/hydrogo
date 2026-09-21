# `hydro dev` Workspace workflow — 2026-09-21

The development entrypoint delegates to the local-only `tools/devserver`
supervisor. The supervisor watches Go, `.templ`, HTML, CSS, JS, YAML and
generated source changes, rebuilds the child HydroGo process, serves bounded
compiler errors, and reloads the browser when the build becomes healthy.

Verification:

```text
go test ./tools/devserver ./internal/server -count=1
ok   github.com/ardaozyildirim/hydrogo/tools/devserver
ok   github.com/ardaozyildirim/hydrogo/internal/server
```

The workflow is development-only. Production `hydro serve` does not start a
watcher, package manager, Templ CLI, or `air`; the single-binary runtime remains
unchanged.

