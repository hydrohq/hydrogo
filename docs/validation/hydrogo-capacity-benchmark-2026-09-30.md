# HydroGo capacity benchmark — 2026-09-30

## Summary

This report records a short, single-host capacity exercise for the HydroGo
Standard runtime. It is an empirical snapshot, not a supported capacity
guarantee or production certification.

An updated run was collected after commits `3dd78a18` and `ee7ccdf8` were
present in the workspace. The new version raised the best observed warm-cache
throughput from about 832 to 1,016 successful requests/s, and dynamic HTML from
about 515 to 560 successful requests/s. The update also introduces explicit
HTTP admission shedding: at overload, excess requests receive HTTP 503 with
`Retry-After` rather than waiting indefinitely in the application. These are
short follow-up samples, so compare trends rather than treating the exact
percentages as guaranteed.

A second follow-up, after commit `372dbbfd` raised the admission thresholds
and exempted `/healthz`/`/readyz`/`/metrics` from shedding, found the warm-cache
peak moved to about 1,414 successful req/s and the dynamic peak to about 668
successful req/s, while HTTP 503 counts at 320 clients dropped from the tens
of thousands (first update, threshold 150) to double digits (this update,
threshold 300). See "Second updated run" below.

A third follow-up, through commit `c012ce09`, added a controlled before/after
(same host, same method, back-to-back) rather than comparing against an
earlier report's numbers: warm cache at 320 clients went from 827.8 to
1,480.3 successful req/s (+79%) while **average process CPU fell** from ~153%
to ~120%, and dynamic HTML at 80 clients went from 462.1 to 729.3 (+58%)
while average CPU fell from ~121% to ~88%. This round also found that every
earlier CPU profile in this report had only exercised gzip, never the Brotli
path real browsers use; profiling with Brotli explicitly negotiated led to
lowering its compression quality from 4 to 1. Separately, an experimental
Caddy edge cache measured 6,184.3 req/s with HydroGo's own CPU near 0%, but
was rejected after it was found to replay the identical CSP nonce to every
client, defeating ADR-0329's per-response freshness guarantee — nothing from
that experiment was deployed. See "Third updated run" below for all details.

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

## Updated run after performance changes

The following follow-up was run on the same server, with the same Standard
profile, routes, separate benchmark database/cache containers, and external
closed-loop Go load client described above. Each stage lasted approximately 12
seconds. The current generator reports successful response latency separately
from overload 503 latency; only successful responses count toward useful RPS.

### Updated warm HTML cache

| Clients | Successful req/s | Successful p50 | Successful p95 | Successful p99 | HTTP 503 | Notes |
|---:|---:|---:|---:|---:|---:|---|
| 80 | 972.3 | 79 ms | 113 ms | 137 ms | 0 | Repeat run |
| 160 | 921.3 | 149 ms | 334 ms | 528 ms | 5 | Near admission limit |
| 320 | 897.3 | 193 ms | 424 ms | 572 ms | 27,913 (71.8%) | Excess load shed quickly |
| 640 | 740.8 | 230 ms | 513 ms | 705 ms | 90,097 (90.9%) | Most attempts rejected |
| 1,000 | 601.7 | 283 ms | 686 ms | 861 ms | 141,052 (95.1%) | Most attempts rejected |

A separate first pass reached 1,016.1 successful req/s at 160 clients, with
12,351 successes and no errors. The repeat at 80 clients reached 972.3 req/s
with p95 113 ms. Across the runs the new peak is roughly 22% above the old
832 req/s peak, and the 80-client point is roughly 20% above the old 808 req/s
point. The run-to-run spread (including a 921 req/s repeat at 160 clients)
shows that the short external test has meaningful variance.

### Updated dynamic HTML (cache bypassed)

| Clients | Successful req/s | Successful p50 | Successful p95 | Successful p99 | HTTP 503 | Notes |
|---:|---:|---:|---:|---:|---:|---|
| 20 | 257.9 | 65 ms | 147 ms | 210 ms | 0 | |
| 40 | 481.5 | 79 ms | 121 ms | 171 ms | 0 | |
| 80 | 548.6 | 140 ms | 207 ms | 265 ms | 0 | Repeat run |
| 160 | 559.7 | 270 ms | 452 ms | 549 ms | 23 | Near admission limit |
| 320 | 487.0 | 340 ms | 566 ms | 684 ms | 33,189 | Most attempts rejected |
| 640 | 355.3 | 415 ms | 908 ms | 1.26 s | 83,892 | Most attempts rejected |
| 1,000 | 315.7 | 519 ms | 869 ms | 1.05 s | 150,837 | Most attempts rejected |

The 80-client dynamic point improved from 467 to 549 successful req/s (about
17.5%), while p95 fell from 251 to 207 ms (about 18%). At 160 clients,
successful throughput increased from 491 to 560 req/s (about 14%) and p95 fell
from 519 to 452 ms (about 13%). The observed dynamic peak increased from 515 to
560 req/s (about 9%), but it occurred at fewer clients and lower p95 latency.
At 20 clients the new result was effectively flat versus the earlier run.

The HTTP admission limit for the Standard profile is 150 in-flight requests.
Therefore 503 counts at 320 clients and above are expected overload responses,
not successful content delivery or client timeouts. The closed-loop generator
immediately retries after each response, so at high client counts it produced
thousands of attempts per second. That artificial retry pressure also consumes
some CPU and means the decreasing successful RPS at high concurrency should not
be read as the service's normal-load throughput. A sensible use of these
results is to assess the admitted requests' throughput/latency and the fact
that excess work is shed promptly; do not count 503 responses as served pages.

### Updated resource sample

| Updated load snapshot | HydroGo process | PostgreSQL container | Redis container | Host memory available |
|---|---:|---:|---:|---:|
| Warm cache, 320 clients | RSS about 73 MiB; CPU about 159% (1.59 logical CPUs) | RSS about 44 MiB; Docker CPU sample about 23% | about 6.5 MiB | about 448 MiB |
| Dynamic HTML, 20 clients | RSS about 53 MiB; CPU about 104% (1.04 logical CPUs) | RSS about 46 MiB; Docker CPU sample about 69% | about 7 MiB | about 469 MiB |

The warm-cache snapshot was collected at high concurrency. The dynamic resource
snapshot was taken at only 20 clients, so it is not a dynamic saturation or
high-water measurement and should not be compared directly with the earlier
320-client dynamic snapshot. No continuous resource collector was active across
all updated stages. The host had 2 vCPUs and 908 MiB visible RAM; the updated
samples showed substantial memory headroom, while the high-load process used
around 1.6 of the 2 logical CPUs. These samples support CPU/queue pressure as a
more immediate limit than RAM for this short workload, but they do not establish
a long-duration memory or CPU ceiling.

## Second updated run after admission threshold revision (commit 372dbbfd)

The first updated run above (Standard profile admission limit = 150 in-flight
requests) was found to undershoot real sustainable concurrency on the dynamic
path: at 320 clients it completed only 487.0 successful req/s while shedding
33,189 requests with HTTP 503, well below the 515.3 req/s the original,
unthrottled baseline completed at the same concurrency with zero errors.
Commit `372dbbfd` raised the thresholds (Compact 32→64, Standard 150→300,
Distributed 600→1,200) and added a bypass so `/healthz`, `/readyz`, and
`/metrics` are never shed even while the limiter is actively rejecting other
traffic — the earlier version had no such bypass, so a sustained overload
could have made an orchestrator conclude the instance itself was unhealthy
and kill or recycle it.

This round used the same host, Standard profile, and isolated benchmark
Postgres/Redis containers as both earlier rounds. The load client ran on a
separate Windows workstation (not on the server) against the server's public
IPv4 address on port 80, using the same closed-loop, keep-alive, no-think-time
model, now also reporting HTTP 503 counts separately from successful-response
latency. `insaat-api` and its PostgreSQL/Redis containers were stopped for the
duration and restarted afterward; a post-test check returned HTTP 401 at `/`,
matching the pre-test state.

### Warm HTML cache (threshold = 300)

| Clients | Successful req/s | p50 | p95 | p99 | HTTP 503 | Failed |
|---:|---:|---:|---:|---:|---:|---:|
| 20 | 350.7 | 50.9 ms | 100.9 ms | 214.9 ms | 0 | 0 |
| 40 | 656.5 | 57.9 ms | 95.1 ms | 122.5 ms | 0 | 0 |
| 80 | 1,093.1 | 68.8 ms | 112.1 ms | 137.0 ms | 0 | 0 |
| 160 | 1,413.6 | 104.6 ms | 188.8 ms | 241.2 ms | 0 | 0 |
| 320 | 1,409.0 | 199.4 ms | 450.8 ms | 606.2 ms | 22 | 0 |
| 640 | 1,154.7 | 278.1 ms | 651.7 ms | 867.6 ms | 54,023 | 0 |
| 1,000 | 965.6 | 362.7 ms | 974.2 ms | 2.21 s | 101,471 | 0 |

### Dynamic HTML, cache bypassed (threshold = 300)

| Clients | Successful req/s | p50 | p95 | p99 | HTTP 503 | Failed |
|---:|---:|---:|---:|---:|---:|---:|
| 20 | 279.1 | 60.6 ms | 140.8 ms | 232.1 ms | 0 | 0 |
| 40 | 409.3 | 88.8 ms | 147.7 ms | 193.7 ms | 0 | 0 |
| 80 | 655.8 | 117.4 ms | 170.7 ms | 220.4 ms | 0 | 0 |
| 160 | 671.5 | 226.5 ms | 371.3 ms | 458.9 ms | 0 | 0 |
| 320 | 668.4 | 449.7 ms | 775.6 ms | 954.1 ms | 85 | 0 |

Note the `fail=0` column in both tables: across every stage in this round,
including the 1,000-client warm-cache run, no request reached the 5-second
client timeout. The original baseline had 6 timeouts at 1,000 clients
(warm cache) and 9 at 1,000 clients (dynamic); with admission shedding in
place, overload now produces fast, cheap HTTP 503 responses instead of
requests that hang until they time out.

### Health-check bypass, verified under active shedding

During the 1,000-client warm-cache run above (947.9 successful req/s,
87,255 HTTP 503 responses over that specific sub-run), five sequential
`/healthz` requests were issued from the load-testing workstation while the
overload run was in flight. All five returned HTTP 200 (100 ms–1.4 s,
network-bound), confirming the bypass added in commit `372dbbfd` keeps
health checks reachable while the admission limiter is actively shedding
other traffic.

### Comparison across all three rounds, low-latency reference point (80 clients)

| Round | Warm-cache req/s | Warm p95 | Dynamic req/s | Dynamic p95 |
|---|---:|---:|---:|---:|
| Original baseline | 807.5 | 149.9 ms | 467.0 | 251.4 ms |
| First update (threshold 150) | 972.3 | 113 ms | 548.6 | 207 ms |
| Second update (threshold 300) | 1,093.1 | 112.1 ms | 655.8 | 170.7 ms |

### Comparison at 320 clients, where the first threshold undershot capacity

| Round | Warm-cache successful req/s | Warm HTTP 503 | Dynamic successful req/s | Dynamic HTTP 503 |
|---|---:|---:|---:|---:|
| Original baseline (no admission limit) | 733.2–753.9 | 0 (no limiter existed) | 515.3 | 0 (no limiter existed) |
| First update (threshold 150) | 897.3 | 27,913 (71.8%) | 487.0 | 33,189 |
| Second update (threshold 300) | 1,409.0 | 22 | 668.4 | 85 |

Raising the threshold did not just restore the pre-limiter throughput at 320
clients — it exceeded it, because the same commits also carry the GC tuning,
faster gzip level, and cache-miss coalescing from commit `3dd78a18`/`ee7ccdf8`.
Shedding at 320 clients is now negligible (22 and 85 requests, versus tens of
thousands under the first threshold); it becomes material again at 640 and
1,000 clients, which is the intended behavior — the limiter is sized to absorb
normal traffic spikes and only shed once concurrency moves well past what
this host can sustain.

As with the first update, part of the very high 503 counts at 640–1,000
clients reflects the closed-loop load client's immediate retry on any
response, including a 503; a real client population would not retry that
aggressively, so the true proportion of shed traffic under equivalent organic
load is likely lower than these raw counts suggest.

## Third updated run: controlled before/after, Brotli profiling, multi-route check, and an edge-cache experiment

This round adds four things the earlier rounds did not have: (1) a true
controlled A/B — the pre-change binary (commit `f1535fd3`, via a separate git
worktree) and the fully updated binary (commit `c012ce09`, everything through
the Brotli quality change) were built and load-tested with the *identical*
method, same host, same stage back-to-back, so the comparison below is not
cross-report; (2) CPU profiling with `Accept-Encoding: br` explicitly set,
because every earlier profile in this report only exercised gzip (Go's
`net/http` client only auto-negotiates gzip, and real browsers prefer
Brotli — CompressionMiddleware picks `br` whenever a client offers it); (3) a
multi-route load test across the demo dataset's 5 distinct news articles; and
(4) an experimental Caddy + Souin edge-cache layer in front of HydroGo, which
surfaced a real security regression described below.

### Controlled before/after (same host, same method, back-to-back)

| Path | Metric | Before (`f1535fd3`, no tuning) | After (`c012ce09`, GOGC=300 GOMEMLIMIT=700MiB, Brotli negotiated) | Change |
|---|---|---:|---:|---:|
| Warm cache, 320 clients | Successful req/s | 827.8 | 1,480.3 | **+79%** |
| Warm cache, 320 clients | p95 | 808.7 ms | 449.5 ms | **-44%** |
| Warm cache, 320 clients | Process CPU (avg / peak) | ~153% / 170% | ~120% / 130% | **-22% avg** |
| Warm cache, 320 clients | Process RSS | 76–79 MiB | 99–278 MiB (spikes with `GOMEMLIMIT`, settles ~100–135 MiB) | higher, by design |
| Dynamic, 80 clients | Successful req/s | 462.1 | 729.3 | **+58%** |
| Dynamic, 80 clients | p95 | 271.3 ms | 147.3 ms | **-46%** |
| Dynamic, 80 clients | Process CPU (avg / peak) | ~121% / 130% | ~88% / 100% | **-27% avg** |
| Dynamic, 80 clients | Process RSS | 60–70 MiB | 77–79 MiB | modestly higher |

Both the "before" and "after" req/s figures land close to their respective
earlier-round measurements in this report (827.8 vs. the original 832 peak;
462.1 vs. the original 467.0 at 80 clients), which cross-validates the
methodology across rounds. The "after" figures here are higher than the
second-round figures at the same concurrency because this run explicitly
negotiated Brotli — the encoding real browsers actually use — instead of
gzip-only.

The headline result: **throughput went up substantially while average CPU
went down**, not merely "throughput per CPU-percent" — actual CPU utilization
dropped in both tests. RSS moved the other way, which is the intended,
accepted trade from raising `GOMEMLIMIT` (see below and the GOMEMLIMIT
experiment write-up already in this conversation's history): more spare host
RAM is now used to let the garbage collector run less often.

### Brotli CPU profiling (new finding this round)

Every CPU profile earlier in this document was taken with a load client that
only sends `Accept-Encoding: gzip` (Go's `net/http` default). Real browsers
send `Accept-Encoding: br, gzip` and prefer Brotli, which
`CompressionMiddleware` honors. Profiling with Brotli explicitly negotiated
showed Brotli quality 4 (the setting in place through the second updated run)
spending the majority of sampled CPU in
`andybalholm/brotli.createBackwardReferences` and related LZ77/context work —
substantially more than gzip's `BestSpeed` level ever cost. Lowering Brotli
to quality 1 (commit `c012ce09`) measured, at 100 clients on this host:

| Brotli quality | Successful req/s | p95 |
|---|---:|---:|
| 4 (prior setting) | 918.0 | 238.9 ms |
| 1 (current setting) | 1,430.8 | 152.8 ms |

Compressed size impact on a real cached page (9,947 bytes uncompressed):
quality 1 produces 3,317 bytes versus gzip `BestSpeed`'s 3,268 bytes for the
same page — a negligible bandwidth cost for the CPU headroom gained. This
finding came directly from testing with the encoding real traffic uses,
which no earlier round in this report had done.

### Multi-route cache check

The demo dataset seeded on this host has only 5 distinct news articles, all
well under even the original (pre-`372dbbfd`) fixed cache limit of 128
entries — so this benchmark cannot exercise the capacity-scaled cache sizes
from ADR-0334 under real eviction pressure. What it can and does confirm:
warming and then load-testing all 5 distinct article routes simultaneously
(160 clients, random selection per request) served 1,307.7 successful req/s
with zero errors or unexpected shedding, confirming the cache correctly
tracks multiple independent keys under concurrent load. Validating the
larger ADR-0334 budgets under genuine eviction pressure would require a much
larger seeded content set than this demo dataset provides; that remains open.

### Edge-cache experiment (Caddy + Souin): real throughput, but not safe as configured

A custom Caddy build (via `xcaddy`, adding the `github.com/darkweak/souin`
HTTP cache module) was placed in front of HydroGo, with HydroGo moved to
`127.0.0.1:8080` and Caddy listening on `:80`. HydroGo's existing
`Cache-Control: public, max-age=<ttl>` header (already emitted by the
ADR-0329 cache on a hit) was sufficient for Souin to cache responses with no
further configuration. The measured effect was dramatic:

| Metric | Direct to HydroGo | Through Caddy + Souin |
|---|---:|---:|
| Warm-cache req/s, 320 clients | 1,480.3 | **6,184.3** |
| p95 | 449.5 ms | **65.6 ms** |
| HydroGo process CPU during the run | ~120–130% | **~0%** (Caddy served every hit) |

HydroGo's own request log confirmed it received zero additional requests
while Caddy served repeated cache hits — the edge cache fully offloaded
warm-cache traffic away from the application process, exactly as intended.

**This configuration was not adopted, because it broke a security guarantee.**
ADR-0329's cache deliberately stores HTML *before* compression and
regenerates the CSP script/style nonce fresh on every response specifically
so a nonce is never replayed to more than one client. Caddy/Souin, sitting in
front of HydroGo, instead cached HydroGo's final compressed bytes —
nonce and all — and replayed the identical `Content-Security-Policy` header,
including the identical nonce value, to every client for the life of the
cache entry. This was verified directly: three consecutive requests through
the Caddy layer returned the exact same
`nonce-Yh1S4aFAg9r1+7X6zWq+gQ==` value. A generic HTTP cache sitting in front
of a nonce-based CSP scheme cannot preserve ADR-0329's per-response
freshness guarantee without cache-side logic that understands and rewrites
the nonce on every replay — which Souin, as a generic RFC 7234 cache, does
not have.

**Disposition:** the Caddy/Souin layer was torn down; nothing from this
experiment was deployed or persisted in the HydroGo binary or its
configuration. This is recorded here as a rejected direction, not an accepted
one. A safe version of edge-level caching would need one of: (a) excluding
any nonce-bearing response from the edge cache (which, on current evidence,
is effectively all public HTML responses, eliminating most of the benefit),
(b) moving to hash-based CSP (`'sha256-...'`) for inline scripts/styles that
are static at build time, which doesn't require per-response freshness and
would be safely cacheable as-is, or (c) a purpose-built cache module that
replicates HydroGo's own nonce-regeneration logic at the edge. Each is a
distinct architectural decision needing its own ADR before implementation;
none is in place today.

## Code changes represented by the updated run

The updated binary included these already committed changes:

- Commit `3dd78a18` adds opt-in pprof diagnostics bound to loopback, sets a
  PostgreSQL connection maximum lifetime, gzip compression at `BestSpeed`, and
  capacity-profile HTTP concurrency limits: Compact 32, Standard 150, and
  Distributed 600 in-flight requests. When the limit is full, HydroGo responds
  with HTTP 503 and `Retry-After`. See
  [ADR-0333](../architecture/0333-http-admission-concurrency-limit.md) for the decision.
- Commit `ee7ccdf8` coalesces concurrent misses for the same public HTML cache
  key using `golang.org/x/sync/singleflight`. A test verifies that eight
  concurrent requests render once while each response retains its own CSP
  nonce; the test was run five times and passed each time. This builds on the
  bounded local HTML cache decision in [ADR-0329](../architecture/0329-bounded-public-html-l1.md).
- These benchmark results do not isolate the individual effect of each code
  change. They compare the combined updated binary with the earlier baseline.

The second updated run adds one more committed change:

- Commit `372dbbfd` raises `Budget.MaxInFlightHTTP` (Compact 32→64,
  Standard 150→300, Distributed 600→1,200) after the first update's Standard
  threshold was found, on this same host, to shed enough dynamic-path traffic
  at 320 clients to complete less successful work (487.0 req/s) than the
  unthrottled original baseline (515.3 req/s) at the same concurrency. It also
  adds a bypass so `/healthz`, `/readyz`, and `/metrics` are never subject to
  the admission limit, closing a gap where a sustained overload could have
  made an orchestrator kill the instance the limiter was trying to protect.
  See the revision note in
  [ADR-0333](../architecture/0333-http-admission-concurrency-limit.md).

The third updated run adds two more committed changes:

- Commit `cd43c257` wires the ADR-0329 public HTML cache's size to
  `capacity.Budget` instead of a single fixed size for every profile
  (Compact 128 entries/16 MiB, Standard 512/128 MiB, Distributed 2,048/512 MiB
  — see [ADR-0334](../architecture/0334-capacity-scaled-public-html-cache-budget.md)).
  This benchmark's 5-article demo dataset never approached the *old* fixed
  limit, so it cannot demonstrate this change's benefit under real eviction
  pressure; see the multi-route check above.
- Commit `c012ce09` lowers Brotli compression quality from 4 to 1, after
  profiling with Brotli explicitly negotiated (rather than this report's
  earlier gzip-only profiles) showed it consuming substantially more CPU per
  request than gzip at a comparable compression ratio. See the Brotli
  profiling section above.

## Interpretation and limits

- On this specific 2-vCPU / 908-MiB host, observed peak throughput was about
  **800 requests/s for the warm public HTML cache** and **500 requests/s for
  dynamic HTML**. Those peak values came with materially higher tail latency as
  concurrency increased.
- In the follow-up run after `3dd78a18` and `ee7ccdf8`, observed peaks were
  about **1,016 successful req/s for warm cache** and **560 successful req/s
  for dynamic HTML**. At 80 clients, the follow-up delivered about 972 req/s
  cached (p95 113 ms) and 549 req/s dynamic (p95 207 ms). Excess requests
  above the admission budget were rejected with HTTP 503 and are excluded from
  successful throughput.
- Before the update, lower-latency reference points were approximately **808
  req/s, p95 150 ms** for warm cache at 80 clients, and **467 req/s, p95 251
  ms** for dynamic HTML at 80 clients. They are short-test reference points,
  not an SLA.
- In the second follow-up run after `372dbbfd` raised the admission
  thresholds, observed peaks rose again to about **1,414 successful req/s for
  warm cache** (at 160–320 clients) and **671 successful req/s for dynamic
  HTML** (at 160 clients). At 320 clients — the concurrency where the first
  threshold (150) undershot capacity — HTTP 503 counts fell from the tens of
  thousands to 22 (warm cache) and 85 (dynamic), while every stage in this
  round completed with zero client timeouts, including at 1,000 clients. This
  round also confirmed `/healthz` stays reachable (HTTP 200) while the limiter
  is actively shedding other traffic, closing the gap noted in the previous
  bullet's admission-shedding behavior.
- The third follow-up run is the first controlled before/after in this
  report: the pre-change binary and the fully updated binary (through
  `c012ce09`) were built and tested with the identical method, back-to-back,
  on the same host. Warm cache at 320 clients went from 827.8 to 1,480.3
  successful req/s (+79%) while **average process CPU fell** from ~153% to
  ~120%; dynamic HTML at 80 clients went from 462.1 to 729.3 successful req/s
  (+58%) while average CPU fell from ~121% to ~88%. Process RSS rose (76–79
  MiB to 99–278 MiB on the cache path), which is the deliberate trade from
  raising `GOMEMLIMIT` to 700 MiB — more of this host's spare RAM is now used
  to reduce GC frequency. This run also found and fixed a real profiling gap:
  every earlier CPU profile in this report only exercised gzip, because the
  load client never negotiated Brotli the way real browsers do; profiling
  with Brotli explicitly requested is what led to the quality 4→1 change in
  `c012ce09`.
- An experimental Caddy + Souin edge cache in front of HydroGo measured
  6,184.3 successful req/s (versus 1,480.3 direct) with HydroGo's own CPU at
  ~0% during the run, but was **not adopted**: it cached HydroGo's
  already-compressed response including its CSP nonce, and replayed the
  identical nonce to every client for the life of the cache entry, defeating
  ADR-0329's per-response nonce-freshness guarantee. This is recorded as a
  rejected direction, not an accepted architecture change; nothing from it
  was deployed or persisted in the codebase.
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

The second updated run (commit `372dbbfd`) followed the same isolation
procedure: `insaat-api`, `insaat-postgres`, and `insaat-redis` were stopped,
the isolated `hydro-bench-postgres`/`hydro-bench-redis` containers were
started, and the updated binary was run as root on port 80 (required to bind
a privileged port) with `GOGC=300` and `GOMEMLIMIT=256MiB`. After the load
stages completed, the HydroGo process was stopped, the benchmark containers
were stopped, and `insaat-postgres`, `insaat-redis`, and `insaat-api` were
restarted. A post-test check against `/` returned HTTP 401, matching the
pre-test state.

The third updated run followed the same isolation procedure across several
back-to-back stages (baseline binary, updated binary, and the Caddy edge-cache
experiment), each stopping `insaat-api`/`insaat-postgres`/`insaat-redis` and
starting the isolated benchmark containers, then reversing that at the end of
the round. The pre-change baseline binary was built from a separate `git
worktree` checked out at commit `f1535fd3` (the last commit before this
report's code changes began) so it could be compiled and load-tested with
the exact same method as the updated binary. The Caddy build (`xcaddy` plus
the `github.com/darkweak/souin` module) required redirecting `TMPDIR` and
`GOTMPDIR` away from this host's small `tmpfs`-backed `/tmp` (only ~91 MiB
free) to a directory on the root filesystem, and required freeing
accumulated benchmark binaries and logs from earlier rounds first, as the
root filesystem was down to under 700 MiB free at one point during this
session. The worktree, the Caddy build tree, and all stray benchmark
binaries/logs on the server were removed at the end of the round. A final
post-test check against `/` returned HTTP 401, matching the pre-test state.

The source adjustment required for PostgreSQL startup removed an empty-string
comparison from the PostgreSQL ban query while retaining that comparison in the
SQLite query. The targeted `go test ./internal/security/sqlstore` and Linux
build had passed before this report was committed. The load results themselves
were collected with the external Go load client described above.
