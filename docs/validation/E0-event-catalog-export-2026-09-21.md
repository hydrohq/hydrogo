# E0 event catalog export validation

- **Date:** 2026-09-21
- **Scope:** Published Core event vocabulary artifact and CLI boundary.

`hydro events catalog` emits the canonical JSON document from
`internal/eventcatalog`. The document has an explicit `v1` version and an
ordered event list; no payloads, endpoint URLs, delivery secrets or consumer
subscriptions are exported. Tests prove the output is deterministic and equals
the in-process catalog. `CompareBaseline` rejects removal and PII classification
changes for published topic/version pairs.

```text
go test ./internal/eventcatalog ./cmd/hydro -count=1
```

Result: pass. Consumer delivery retry/dead-letter rehearsal remains open.
