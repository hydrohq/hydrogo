# R5 portable export/import validation

**Date:** 2026-09-09  
**Roadmap:** R5 / S09  
**Status:** Allowlist-based SQLite round-trip succeeded; PostgreSQL round-trip CI contract and large SQLite fixture added. Reproducible benchmark command added; comparative benchmark with customer hardware remains open.

`internal/portable` carries pages, page revisions, media metadata, Site Edition, form inbox, and News records in a deterministic JSON format. Identity, revision, locale, and relationships are preserved; `users`, `sessions`, MFA/recovery/invite tokens, audit, outbox, and jobs tables are not included in the format allowlist. `ExportWithLimit` applies a byte limit while reading rows and creating a partial document; the CLI `--max-bytes` check is not performed after the document has fully grown.

Executed check:

```text
go test ./internal/portable
```

For the memory limit, the `TestExportWithLimitRejectsOversizedRowsDuringRead` test
was added and validated in the `fix(portable): bound export memory during reads` (`70bbc43`)
commit. Full validation is also successful via `go test ./... -p 1`,
`go vet ./...` and `git diff --check`.

The contract test takes an export from a source SQLite and imports it to a separate SQLite target; page and revision are preserved, and it is validated that auth state does not leak out. A fixture of 256 page/revision/media records verifies identity, locale, revision, and relationships post-round-trip. Under the CI condition with `HYDRO_TEST_POSTGRES_URL` defined, the same API performs a source/target round-trip on temporary PostgreSQL schemas; the test is skipped locally.

`BenchmarkSQLitePortableRoundTrip256` measured 15.4 ms/round-trip on the last run with `-benchtime=10x` on a Windows amd64 development machine (Intel(R) Core(TM) 7 240H), with the target schema prepared before the benchmark starts. Significant volatility (15.4–369.9 ms/op) was observed between runs on the same machine; therefore, the result is not proof of deterministic performance or SLA. The measurement covers export + JSON encode/decode + in-transaction import to the existing target; SQLite startup and migration setup are out of scope. It should be re-measured under load on pilot customer hardware and tied to RPO/RTO and delivery times.
