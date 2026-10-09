# Generated Admin record read denial — 2026-10-10

The shared Admin's record URL first loads the record through the Edition API.
Previously, every non-200 response used the service-unavailable retry page.
That mislabeled a record-level API authorization denial as an outage and did
not match the Admin's existing concealed-resource behavior.

The record route now maps Edition API `403` and ordinary-collection `404` to
Admin `404`. Other failures, including `503`, continue to use the sanitized
`502` retry page. Draft collections preserve their existing API `404` behavior
for unpublished draft identities. The Edition API remains the read-policy
authority; no policy is inferred from Blueprint metadata and no database
behavior changes.

`TestGeneratedAdminConcealsRecordReadDenialButKeepsServiceRetry` verifies API
`403` and `404` conceal the Admin detail, a `503` retains the retry affordance,
and no upstream response body is exposed.

```sh
GOCACHE=/tmp/hydro-cover-cache go test ./internal/adminstudio \
  -run '^(TestGeneratedAdminConcealsRecordReadDenialButKeepsServiceRetry|TestGeneratedAdminResourceReadPolicyHidesIndexAndBlocksCollectionRoutes|TestGeneratedAdminLocalizesAndSanitizesResourceFailure)$' \
  -count=1 -p 1
GOCACHE=/tmp/hydro-cover-cache go test ./internal/adminstudio ./internal/i18n \
  -count=1 -p 1
GOCACHE=/tmp/hydro-cover-cache go vet ./internal/adminstudio ./internal/i18n
```

All commands passed. This is handler-contract evidence; the independent
Edition's real browser and SQLite/PostgreSQL deny/readback tests remain the
evidence for its API policy and persisted state.
