# R102 — HydroGo Golden-Path Two Resource Baseline

**Date:** 2026-09-22  
**Status:** HydroGo baseline successful; Payload runtime smoke measurements
collected. Full parity comparison is not yet completed.

## Scenarios

`scripts/measure-iteration.ps1 -Runs 3` was executed in the same Windows/Go environment with
the following two independent resource definitions:

| Resource | Fixture | Median | Min–max | Acceptance |
| --- | --- | ---: | ---: | --- |
| Testimonial | `examples/module-starter/collection.yaml` | 1,651 ms | 1,620–9,137 ms | PASS |
| Article | `examples/benchmark/article-collection.yaml` | 1,546 ms | 1,498–1,551 ms | PASS |

In each scenario, the measured flow includes project scaffold, route manifest verification, resource
generation, and generated compatibility check steps. Each of the three runs
remained below the 15-minute local acceptance threshold.

## Commentary

This report serves as a reproducible DX baseline for two HydroGo resource scenarios.
This table alone does not constitute a Payload parity claim. The 9,137 ms cold-start outlier
in the initial testimonial run should be monitored separately; warm runs
fall within the 1,620–1,651 ms range.

## Payload Smoke Evidence

On the same Windows machine, a temporary SQLite-based application was installed via PTY using
`create-payload-app@3.90.1 -t blank`. For the `Articles` collection, an
`Articles.ts` file and a `payload.config.ts` entry were added manually; this exposes
the additional touches required for an authentic Payload collection compared to a HydroGo blueprint.
`npm run generate:types` and `npm run build` succeeded.

| Metric | Result |
| --- | ---: |
| Payload / Next | 3.90.1 / 16.3.3 |
| Initial `/admin` response | HTTP 200, approximately 13.7 s (cold compile) |
| Warm `/admin` | 61–163 ms |
| Warm `GET /api/articles?limit=10` | HTTP 200, 114–180 ms |
| Warm production build | 16,856 ms |
| Application source files | 17 |
| `node_modules` size | approximately 701.6 MB |

In the first attempt without PTY, the CLI reported `uv_tty_init ... EBADF`; with PTY, setup
completed successfully. This demonstrates that the test harness blocker was resolved. Still,
the measurement does not represent full parity: authenticated admin CRUD/browser scenarios,
seeded data, p95/p99 load testing, and equivalent feature scope have not yet been compared.
The initial API request returned 403 until the access rule hot-reloaded; subsequently, public read
returned 200. This observation necessitates separate evaluation of Payload's default access/configuration overhead.

The temporary Payload workspace was retained under Temp outside the repository after measurement;
it is not a release artifact.

Raw outputs were generated under `.tmp/benchmark-module-starter.json` and
`.tmp/benchmark-article-collection.json`; `.tmp` is not a release artifact.
