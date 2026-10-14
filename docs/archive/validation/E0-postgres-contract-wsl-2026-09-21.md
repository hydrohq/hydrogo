# E0 PostgreSQL Contract Certification — 2026-09-21

Environment: WSL2 Ubuntu, PostgreSQL 18.x, Go 1.27.0.

Command:

```text
HYDRO_TEST_POSTGRES_URL="postgres:///hydrogo_cert_20260921?host=/var/run/postgresql" \
  go test ./tests/contract ./internal/tenant -count=1
```

Observed result: PASS. The run covers transaction, pages/content, news,
authentication, media lifecycle, site edition, tenant provisioning, module
filtering and PostgreSQL RLS isolation contracts. The database was a fresh
isolated certification database.

The RLS migration was hardened to skip tables that are intentionally absent
from filtered module profiles; this preserves inactive-module isolation while
keeping the policy fail-closed on tables that are present.
