# News taxonomy relational contract — 2026-09-16

The News-owned taxonomy migration and repository were exercised against both
SQLite and a real PostgreSQL 18.6 service.

## Evidence

```text
Windows: go test ./tests/contract -run 'NewsTaxonomy' -count=1       PASS
WSL:     HYDRO_TEST_POSTGRES_URL=... go test ./tests/contract \
         -run 'NewsTaxonomy' -count=1                                  PASS
```

The contract creates and updates localized category/tag terms, replaces content
relations, reads deterministic results, discovers bounded related published
content, and clears relations. Repository tests also verify that parent terms
with children and terms still referenced by content cannot be deleted. Migration
filtering proves the News taxonomy tables are absent when the News module is
disabled.

The PostgreSQL run used a disposable local database and no customer data. It is
local adapter evidence, not a production restore, capacity or external provider
certification.
