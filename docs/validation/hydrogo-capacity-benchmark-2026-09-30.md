# HydroGo capacity benchmark — 2026-09-30

## Summary

This report records a short, single-host capacity exercise for the HydroGo
Standard runtime. It is an empirical snapshot, not a supported capacity
guarantee or production certification.

| Request path | Best observed throughput | Latency at that point | Errors |
|---|---:|---:|---:|
| Public news HTML, warm HydroGo process cache | 832 requests/s at 160 clients | p95 362 ms; p99 473 ms | 0 |
| Public news HTML, cache bypassed with a query string | 515 requests/s at 320 clients | p95 1.03 s; p99 1.27 s | 0 |

Throughput stopped increasing beyond these points. More clients mostly increased
queueing delay; at 1,000 clients, a small number of requests timed out. For a
lower-latency operating point, the cache path served about 808 requests/s at 80
clients (p95 150 ms), while the dynamic path served 467 requests/s at 80
clients (p95 251 ms).

## Test environment

- Host: Ubuntu 26.04, x86-64, **2 vCPU**, **908 MiB RAM visible to Linux**, and
  2 GiB swap. The instance type was not recorded.
- HydroGo: Standard profile, Linux/amd64 binary built from this workspace.
  The build included the PostgreSQL predicate fix in
  [`internal/security/sqlstore/bans.go`](../../internal/security/sqlstore/bans.go).
- HydroGo ran as a host process without a container CPU or memory cgroup limit.
  Binding directly to port 80 meant the temporary benchmark process ran with
  root privileges.
- Data services: isolated PostgreSQL 15 and Redis 7 benchmark containers. Their
  memory limits were 192 MiB and 48 MiB respectively. The active inşaat
  PostgreSQL/Redis containers were not used.
- Content: the isolated benchmark database was seeded with HydroGo demo content;
  this was not a large production dataset.
- Load generator: a separate Windows workstation, using a small Go HTTP client
  with keep-alive, one worker per configured client, no think time, and a
  5-second per-request timeout. Requests went to the server's public IPv4
  address, so the measurement includes the network path and generator.
- The benchmark binary listened on port 80 during the test. No writes or
  authenticated workflows were part of the saturation runs.
- Rate limiting was not disabled. The measured anonymous public GET routes do
  not use the login rate-limit path; the earlier login check had confirmed that
  the Redis-backed login limiter remained active.

The load generator was not instrumented for its own CPU/network ceiling, and the
instance type and AWS credit state were not recorded. Treat the numbers as
observed throughput for this setup, not as a clean-room comparison across
machines.

## Saturation method

Two routes were measured:

1. `GET /public/news/tr/html` — anonymous request, no cookies or query string.
   The first request warmed the process-local public HTML cache; subsequent
   requests could receive the cached HTML.
2. `GET /public/news/tr/html?bench=1` — the same public view with a query string,
   which deliberately bypasses the HTML response cache and exercises dynamic
   rendering and data access.

The test used a closed-loop load model: each client sent another request after
its previous response completed. Each stage ran for about 10–12 seconds. The
client count was raised until throughput stopped increasing and latency or
timeouts worsened. The figures below are successful HTTP responses divided by
elapsed wall time; p50/p95/p99 are end-to-end client latency, including network
round trips. Short stages and network variance can move these values.

## Warm HTML cache results

| Clients | Duration | Requests | Successful | Failed | Successful req/s | p50 | p95 | p99 | Max |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 20 | 12.1 s | 3,996 | 3,996 | 0 | 331.4 | 56.8 ms | 109.5 ms | 207.9 ms | 525 ms |
| 40 | 12.1 s | 5,769 | 5,769 | 0 | 476.5 | 77.2 ms | 134.0 ms | 196.8 ms | 344 ms |
| 80 | 12.1 s | 9,769 | 9,769 | 0 | 807.5 | 93.4 ms | 149.9 ms | 194.8 ms | 330 ms |
| 160 | 12.2 s | 10,137 | 10,137 | 0 | **831.8** | 170.9 ms | 361.5 ms | 473.1 ms | 997 ms |
| 320 | 13.6 s | 9,936 | 9,936 | 0 | 733.2 | 341.1 ms | 988.3 ms | 1.62 s | 2.53 s |
| 640 | 12.7 s | 9,807 | 9,807 | 0 | 773.0 | 678.8 ms | 1.85 s | 2.48 s | 4.44 s |

Additional repetitions:

| Clients | Duration | Requests | Successful | Failed | Successful req/s | p50 | p95 | p99 | Max |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 160 | 12.1 s | 9,608 | 9,608 | 0 | 788.0 | 206.6 ms | 359.0 ms | 388.4 ms | 423 ms |
| 320 | 10.4 s | 7,817 | 7,817 | 0 | 753.9 | 349.6 ms | 933.6 ms | 1.35 s | 2.25 s |
| 640 | 10.7 s | 8,500 | 8,500 | 0 | 792.8 | 670.1 ms | 1.73 s | 2.34 s | 3.88 s |
| 1,000 | 11.7 s | 8,832 | 8,826 | 6 | 753.6 | 1.05 s | 2.68 s | 3.68 s | 5.00 s |
| 640 | 12.7 s | 10,160 | 10,160 | 0 | 798.4 | 667.3 ms | 1.73 s | 2.39 s | 3.67 s |
| 640 | 12.7 s | 10,054 | 10,053 | 1 | 791.8 | 680.3 ms | 1.75 s | 2.36 s | 5.00 s |

The cache-path throughput repeatedly clustered around 750–830 requests/s. At 80
clients, it delivered about 808 requests/s with p95 below 150 ms. Increasing to
160 clients barely increased throughput and more than doubled p95; 320–1,000
clients mostly increased queueing, with timeouts appearing at the highest load.

## Dynamic HTML results (response cache bypassed)

| Clients | Duration | Requests | Successful | Failed | Successful req/s | p50 | p95 | p99 | Max |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 20 | 10.1 s | 2,645 | 2,645 | 0 | 262.9 | 67.2 ms | 135.3 ms | 174.3 ms | 214 ms |
| 40 | 10.4 s | 4,557 | 4,557 | 0 | 437.0 | 82.7 ms | 135.6 ms | 233.3 ms | 462 ms |
| 80 | 10.2 s | 4,741 | 4,741 | 0 | 467.0 | 162.0 ms | 251.4 ms | 320.7 ms | 457 ms |
| 160 | 10.3 s | 5,043 | 5,043 | 0 | 491.0 | 305.9 ms | 518.9 ms | 618.9 ms | 879 ms |
| 320 | 12.5 s | 6,439 | 6,439 | 0 | **515.3** | 577.5 ms | 1.03 s | 1.27 s | 1.82 s |
| 640 | 12.9 s | 6,437 | 6,437 | 0 | 497.6 | 1.17 s | 2.13 s | 2.66 s | 4.68 s |
| 1,000 | 12.3 s | 5,715 | 5,706 | 9 | 463.9 | 1.86 s | 3.30 s | 4.05 s | 5.00 s |

Dynamic throughput topped out around 490–515 requests/s. The 320-client result
had the highest completed throughput, but p95 was already about one second; it
is a saturation result rather than a recommended operating target. At 1,000
clients, nine requests reached the 5-second client timeout.

## Earlier paced measurements

These earlier figures use a different load model and must not be mixed with the
saturation results above. Each virtual user sent at most one request per
second, and the generator ran on the same host as HydroGo. They are useful for
context, not for finding the maximum:

| Route | Clients | Throughput | p50 | p95 | p99 | Responses |
|---|---:|---:|---:|---:|---:|---:|
| `/api/pages?locale=tr` (database-backed API list) | 250 | 263 req/s | 6.2 ms | 84 ms | 107.4 ms | 2,750/2,750 HTTP 200 |
| `/public/news/tr/html?bench=1` (dynamic HTML) | 250 | 256 req/s | — | 132 ms | — | 2,250/2,250 HTTP 200 |
| Same dynamic HTML route, repeat | 250 | 266 req/s | — | 3.9 ms | — | 3,250/3,250 HTTP 200 |

The large spread between the two short HTML repeats shows why the paced,
same-host result should not be treated as a stable capacity figure. These tests
used 10–12 second samples and loopback traffic; the current saturation tests
used an external generator.

## Resource observations

Resource collection was sampled during several stages, not continuously across
every request stage. Values are approximate snapshots, not high-water marks
from a dedicated metrics collector.

| Load snapshot | HydroGo process | PostgreSQL container | Redis container | Host memory available |
|---|---:|---:|---:|---:|
| Dynamic HTML, 320 clients | RSS about 154 MiB; process CPU about 123% (roughly 1.23 logical CPU equivalents) | RSS about 38 MiB; Docker CPU sample about 51% | about 5 MiB | about 335 MiB |
| Warm cache, 320 clients | RSS about 98 MiB; process CPU about 159% (roughly 1.59 logical CPU equivalents) | RSS about 36 MiB; Docker CPU sample about 22% | about 5 MiB | about 380 MiB |

The host began with roughly 474 MiB available after unrelated applications
were stopped. During the test, the lowest captured `MemAvailable` was about 335
MiB. Swap use stayed near its pre-existing 500 MiB level; no increase or OOM
event was observed. Redis stayed below 1% in the sampled Docker CPU readings.
The measurements suggest CPU and queueing—not RAM exhaustion—limited these
short GET runs, but continuous host CPU telemetry was not captured for every
stage. Linux process CPU percentages above 100% mean the process used more than
one logical CPU; container CPU percentages are the Docker CLI readings and are
reported separately.

## Interpretation and limits

- On this specific 2-vCPU / 908-MiB host, observed peak throughput was about
  **800 requests/s for the warm public HTML cache** and **500 requests/s for
  dynamic HTML**. Those peak values came with materially higher tail latency as
  concurrency increased.
- For lower tail latency, the measured points were approximately **808 req/s,
  p95 150 ms** for warm cache at 80 clients, and **467 req/s, p95 251 ms** for
  dynamic HTML at 80 clients. They are short-test reference points, not an SLA.
- A cache hit is not equivalent to an authenticated request, a write, an API
  query, or a cache miss. This report does not measure login, admin, writes,
  media uploads, cold start, long soak, deployment restarts, or multi-process
  invalidation.
- The process-local HTML cache is bounded and has the default 30-second TTL.
  Cache invalidation and multi-process consistency remain subject to
  [ADR-0329](../architecture/0329-bounded-public-html-l1.md) and
  [ADR-0011](../architecture/0011-consistency-upgrade-and-recovery.md).
- The earlier 263 req/s API figure is a separate paced test and is not the
  dynamic HTML saturation result.
- The instance type, generator hardware load, disk I/O profile, real customer
  dataset, and AWS CPU-credit/steal history were not recorded. The results must
  not be extrapolated to another server size or customer traffic mix.

## Service handling and verification record

For isolation, the running inşaat API, its PostgreSQL/Redis containers, and the
host PostgreSQL service were stopped temporarily; they were not deleted. The
benchmark used separate HydroGo-only PostgreSQL/Redis containers. After the
test, the benchmark HydroGo process and its containers were stopped and the
previous services were started again. The inşaat API returned HTTP 401 at `/`,
which was its pre-test observed response.

The source adjustment required for PostgreSQL startup removed an empty-string
comparison from the PostgreSQL ban query while retaining that comparison in the
SQLite query. The targeted `go test ./internal/security/sqlstore` and Linux
build had passed before this report was committed. The load results themselves
were collected with the external Go load client described above.
