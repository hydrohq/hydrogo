# R88 — Headless API compatibility matrix

Date: 2026-09-19

## Evidence

The independent TypeScript headless starter builds and passes its client
contract smoke. Each published v1 contract was compared from its checked-in
baseline (old client expectation) to the current server document:

| Audience | Baseline | Current | Result |
| --- | --- | --- | --- |
| Staff mobile | `docs/api/baselines/mobile-v1.openapi.json` | `docs/api/mobile-v1.openapi.json` | pass |
| Member mobile | `docs/api/baselines/member-mobile-v1.openapi.json` | `docs/api/member-mobile-v1.openapi.json` | pass |
| Member commerce | `docs/api/baselines/member-commerce-v1.openapi.json` | `docs/api/member-commerce-v1.openapi.json` | pass |
| Public content | `docs/api/baselines/public-content-v1.openapi.json` | `docs/api/public-content-v1.openapi.json` | pass |

Commands:

```text
npm test (examples/headless-starter)
go run ./tools/openapidiff --baseline docs/api/baselines/mobile-v1.openapi.json --current docs/api/mobile-v1.openapi.json
go run ./tools/openapidiff --baseline docs/api/baselines/member-mobile-v1.openapi.json --current docs/api/member-mobile-v1.openapi.json
go run ./tools/openapidiff --baseline docs/api/baselines/member-commerce-v1.openapi.json --current docs/api/member-commerce-v1.openapi.json
go run ./tools/openapidiff --baseline docs/api/baselines/public-content-v1.openapi.json --current docs/api/public-content-v1.openapi.json
```

All checks passed. This is repository-level old-client/new-server evidence;
staging with an old running server and the new client, plus reverse-proxy/BFF
cross-origin behavior, remains an external release gate.
