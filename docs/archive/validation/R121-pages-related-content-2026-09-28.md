# R121 — Related page recommendations

The Pages module stores current tags and a published snapshot separately.
Public recommendations are indexed by tenant/tag, locale-scoped, rank by
shared tags, exclude the current page and non-published pages, and cap the
public page-render request at four. The repository contract also verifies
that changing draft tags does not alter public suggestions before publication.

Verification:

```text
go test ./modules/pages/... ./modules/news/repository/sqlstore/... ./modules/portfolio/repository/sqlstore/...
PASS
HYDRO_TEST_POSTGRES_URL=postgres://... go test -p 1 ./modules/site/... ./modules/pages/repository/sqlstore ./modules/news/repository/sqlstore ./modules/portfolio/repository/sqlstore
PASS on PostgreSQL 18.6 (temporary local instance)
go test ./modules/pages/repository/sqlstore -run '^$' -bench BenchmarkListRelatedPublishedSQLite -benchtime=10x
2000 matching pages: 4.88 ms/op
```

The PostgreSQL URL is opt-in and was supplied for the temporary validation
instance only. The SQLite query plan confirms the relation lookup index is
selected. The 4.88 ms result is a local synthetic SQLite measurement, not a
cross-dialect or production SLA. The Pages migration is module-scoped;
inactive Editions receive neither relation table nor indexes.
