# E0 Construction Persistence — 2026-09-21

The external Construction Edition now includes an edition-owned persistence
package and additive SQLite/PostgreSQL migrations. Document approval performs
project-bound role authorization, revision compare-and-swap, and audit append
inside one database transaction.

Validation:

```text
cd examples/construction-edition
go test ./... -count=1
```

The test proves viewer denial, manager approval, stale revision rejection, and
the single audit event. Set `HYDRO_CONSTRUCTION_POSTGRES_URL` to run the same
contract against PostgreSQL.

Scope note: this increment covers the document approval slice. Daily-report
and issue command/repository methods remain a follow-up before E0.9b can be
marked complete.
