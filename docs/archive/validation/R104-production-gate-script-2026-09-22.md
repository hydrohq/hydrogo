# R104 — Production gate runner compatibility

**Date:** 2026-09-22  
**Status:** PASS (script parse/entry gate)

`scripts/verify-production-gates.ps1` was made parseable under both Windows PowerShell 5 and PowerShell 7. Previous Unicode em-dash characters triggered invalid string-terminator errors in Windows PowerShell 5's source code interpreter; messages were normalized to ASCII `-`.

The command now reaches the expected fail-closed check without parser errors:

```text
HYDRO_TEST_POSTGRES_URL is required; PostgreSQL contract tests must not be skipped.
```

Because a PostgreSQL URL was not provided on this machine during this run, the production gate suite was not executed. This is reported not as a gate failure, but as an absence of mandatory external service configuration.
