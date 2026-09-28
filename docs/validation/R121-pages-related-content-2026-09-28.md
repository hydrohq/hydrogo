# R121 — Related page recommendations

The Pages module stores current tags and a published snapshot separately.
Public recommendations are indexed by tenant/tag, locale-scoped, rank by
shared tags, exclude the current page and non-published pages, and cap the
public page-render request at four. The repository contract also verifies
that changing draft tags does not alter public suggestions before publication.

Verification:

```text
go test ./modules/pages/...
PASS
```

The opt-in PostgreSQL contract test requires `HYDRO_TEST_POSTGRES_URL`; it was
not run in this environment. The SQLite query plan confirms the relation
lookup index is selected. The Pages migration is module-scoped; inactive
Editions receive neither relation table nor indexes.
