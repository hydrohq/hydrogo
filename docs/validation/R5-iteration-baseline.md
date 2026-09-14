# R5 validation: developer iteration baseline

The repository now includes `scripts/measure-iteration.ps1`. It measures two
repeatable steps for a standard customer feature loop:

1. `go test ./internal/ui ./tests/contract` (targeted feedback)
2. `go test ./... -run '^$'` (full compile feedback)

Run it from the repository root. Use `-OutputPath` to save a JSON report with
the UTC timestamp, commit, Go version, durations, and exit codes:

```powershell
./scripts/measure-iteration.ps1 -OutputPath .tmp/iteration-baseline.json
```

The result is an engineering baseline for comparing changes on the same
machine, not a universal speed or SLA claim. Customer feature delivery time,
training time, browser QA, and deployment time still require a real pilot.
