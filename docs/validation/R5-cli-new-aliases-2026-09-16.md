# R5 — CLI `new` compatibility aliases

While preserving existing `create` commands for backward compatibility, Payload-style shorthand commands were added:

```text
hydro module new --spec SPEC --out OUTPUT
hydro resource new --spec SPEC --out OUTPUT
hydro resource check --spec SPEC --root OUTPUT
hydro api diff --baseline OPENAPI.json --current OPENAPI.json
hydro check --config config.yaml --json
```

`cmd/hydro` tests passed on Windows and WSL Ubuntu. `resource new --name article --field title:text` smoke test produced deterministic scaffolds; regression tests were also executed for existing `create` commands.
`resource check` confirmed generator compatibility against the same spec and output root.
`api diff` is a compatibility alias bringing OpenAPI compatibility checks into the SDK workflow; the backward-compatible `validate api` command is preserved.
`check` is a top-level SDK shortcut for `project check`.

WSL command:

```text
go test ./cmd/hydro ./internal/generator ./internal/release ./internal/sdkdist
PASS
```
