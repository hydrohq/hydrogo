# Generated Edition SQLite contract — 2026-09-21

`tests/contract/edition_scaffold_test.go` loads the checked-in Project and
Article Edition artifacts, parses their `migration.yaml`, and applies each
generated `migration.sql` to an isolated in-memory SQLite database. The test
also requires additive `expand` metadata for both SQLite and PostgreSQL and
rejects destructive table drops.

```text
go test ./tests/contract -run Edition -count=1
ok   github.com/ardaozyildirim/hydrogo/tests/contract
```

This proves the generated migration artifact boundary. PostgreSQL execution,
full server wiring, and browser journeys remain separate Edition/staging gates.

