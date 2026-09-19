# R89 — CI OpenAPI step wiring

Date: 2026-09-19

## Finding and fix

The four OpenAPI compatibility steps in `.github/workflows/ci.yml` were nested
under the preceding `run` mapping because of an extra indentation level. That
could prevent the GitHub Actions workflow from parsing or executing the API
compatibility gate. The steps are now aligned with the other `go` job steps.

## Evidence

```text
python -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"
PASS: ci.yml YAML parsed
go run ./tools/actionguard
PASS
```

`git diff --check` also passed. The workflow remains read-only and all action
references continue to be checked by the repository action pin guard.
