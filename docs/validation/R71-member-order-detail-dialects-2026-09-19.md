# R71 member order detail dialect contract

**Date:** 2026-09-19  
**Status:** SQLite and WSL PostgreSQL ownership contract passed.

The member order-detail projection is certified against both supported SQL
dialects. The test inserts a paid order and line item, verifies the member can
read its bounded payment/shipping projection, and verifies a different member
receives `sql.ErrNoRows` before projection. No provider credentials or live
payment endpoint is involved.

```text
go test ./tests/contract -run TestSQLiteMemberOrderDetailOwnershipContract -count=1
wsl.exe -e bash -lc 'export HYDRO_TEST_POSTGRES_URL=...; go test ./tests/contract -run TestPostgresMemberOrderDetailOwnershipContract -count=1'
```

The PostgreSQL run used the isolated local `hydro_gate_local` database. This is
database contract evidence, not a production restore, provider sandbox, or
multi-tenant certification.
