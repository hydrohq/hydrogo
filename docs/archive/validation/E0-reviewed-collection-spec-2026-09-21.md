# E0 reviewed collection specification validation

- **Date:** 2026-09-21
- **Scope:** ADR-0200's reviewed specification-to-code boundary only.
- **Not evidence for:** a browser designer, runtime schema modification, or
  automatic deployment approval.

## Evidence

`internal/generator.SpecDocument` fixes the input version at `v0`.
`generator.LoadSpec` accepts only real JSON/YAML files, rejects symbolic links
and unknown fields, and sends the decoded collection through `Validate`.
Validation requires both SQLite and PostgreSQL dialect declarations, explicit
permissions, non-destructive change intent, and an explicit target plus `one`
or `many` cardinality for relation/reference fields.

`generator.Generate` is a deterministic artifact generator. It does not start
an HTTP server, register a route, run a migration, or mutate a live database.
The generated migration remains a review artifact subject to the normal
dialect, compatibility and release workflow.

## Commands run

```text
go test ./internal/generator ./cmd/hydro -count=1
```

Result: pass.

## Follow-up boundary

A future visual editor must receive its own security/authorization decision
and browser coverage. It cannot use this accepted CLI/spec capability as
authority to publish a collection or apply DDL.
