# Payload Parity Benchmark Protocol

This protocol specifies the measurement process for evaluating HydroGo's development velocity
relative to Payload CMS. The goal is not marketing posturing, but collecting reproducible evidence
under identical hardware and test conditions.

## Fixed Test Scenario

Both systems model and implement an `articles` resource:

- `title` (required, localized)
- `slug` (unique index)
- `body` (rich text)
- `cover` (media)
- `status` (draft/published)
- `publishedAt` (scheduled publishing)
- `category` (relation)

The scenario encompasses database migrations, seed data, admin listing/editing interfaces,
publishing workflows, a public detail API endpoint, input validation, and a browser smoke test.
Any hand-coded business logic additions are logged separately.

## Target Metrics

Record the following metrics during each test execution:

1. Elapsed wall-clock time from initial command to working admin and API smoke tests.
2. Count of generated files versus manually modified files.
3. Count of failed commands or tests and required remediation steps.
4. Total CLI command count required to reach initial working state.
5. Cold and warm request latency (p50, p95) and error rates.
6. Peak RSS memory, CPU usage, open database connections, and cache hit metrics.

## Experimental Parameters

- Identical Windows/WSL host environment, pinned Go/Node/Payload versions, and clean checkouts.
- Identical SQLite baseline dataset; followed by an identical supported PostgreSQL version run.
- Cold cache and warm cache results reported independently.
- Heavy external dependencies (payment, S3, CDN) mocked unless provider/region is explicitly noted.
- Each system is executed at least three times; report median and range values.

## HydroGo Baseline Commands

```bash
hydro project new benchmark-news --preset=news
cd benchmark-news
hydro project check
hydro resource create --name articles --field title:text --field slug:text
hydro generate
hydro migrate
hydro test --package ./...
```

Never conceal missing automation or manual wiring requirements; command outputs and execution
timings are preserved in a dated validation report under `docs/validation/`.

For a reproducible local DX baseline, `scripts/measure-iteration.ps1` executes three runs and
accepts a bounded repetition parameter (`-Runs N`, between 1 and 10). The report outputs individual
run durations, median/min/max metrics, and success status. This internal measurement script does
not replace side-by-side comparative benchmarking against the target system.

## Acceptance Criteria

HydroGo can only claim to be "as fast as Payload" when initial working admin/API workflows across
two independent resource scenarios complete within the 30-minute target threshold with measured
manual step and error counts. The 15-minute threshold of the local HydroGo iteration script is a
rapid feedback gate only; it does not replace the 30-minute parity goal. This protocol itself is not
proof of success; until verified by a final benchmark report, parity status remains **open**.
