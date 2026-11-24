# HydroGo Project Context

Short product and positioning brief for people and AI agents. For code layout see
[`architecture/CODEMAP.md`](architecture/CODEMAP.md); for direction see [`VISION.md`](VISION.md).

## 1. Product & strategy

- **Product:** HydroGo (`hydrogo.dev`), built by the studio ArdaLabs (`ardalabs.dev`).
- **What it is:** a single-binary, AI-native headless Go engine: CMS core (pages, news, media,
  members, content types, REST API, embedded Svelte admin) plus provider-agnostic commerce
  contracts (catalog, cart, checkout). See [`decisions/0001-cms-focus.md`](decisions/0001-cms-focus.md).
- **Business model:** open core under PolyForm Shield 1.0.0 (source-available, free to self-host).
  - **Core (this repo, `hydrohq/hydrogo`):** `net/http` engine, PostgreSQL (RLS), SQLite (WAL) and
    MySQL, YAML content-type schemas, embedded HTML/static engine, headless JSON APIs, 503
    fast-fail backpressure.
  - **Commercial (private):** `hydrohq/hydro-modules` (PayTR 3D Secure, iyzico, Geliver/Yurtiçi
    shipping, BizimHesap e-Archive invoices) and `hydrohq/hydrocloud` (managed hosting).

## 2. Stack

- **Engine:** Go (version in `go.mod`), stdlib `net/http` routing, `sync.Pool` buffer reuse
  (`internal/server/http_optimization.go`).
- **Databases:** PostgreSQL with row-level security for tenant isolation, SQLite in WAL mode, MySQL.
- **Edge:** Caddy reverse proxy behind Cloudflare (WAF, DNS, edge cache).
- **AI layer:** MCP server and CLI for natural-language store administration.
- **Storefront:** embedded `embed.FS` HTML or an external SvelteKit / Next.js SSR app.

## 3. Benchmarks (Oracle Cloud ARM64, 4 vCPU, 23 GiB RAM)

Measured once; re-run before quoting publicly.

| Metric | Result |
|---|---|
| Memory | 38–42 MB RSS idle/k6; 186 MB peak under stress with Node SSR |
| `GET /healthz` | 4,925 req/s (p50 5.81 ms) |
| `GET /api/v1/pages` (JSON + SQLite) | 3,198 req/s (p50 8.91 ms) |
| `PublicHTMLCache` | 5,581 req/s (p50 21.1 ms) |
| Writes | 864.3 req/s (p95 99.9 ms), 0% errors, 20/20 SSE sync |
| Backpressure | at c=512 excess load shed as 503 instead of growing memory |

`MaxInFlightHTTP` comes from the capacity profile in `internal/capacity/profile.go`
(64 / 300 / 1200); the run above used the 300 profile.

## 4. Brands & domains

- `ardalabs.dev`: engineering studio (custom systems, integrations, backend consulting).
- `hydrogo.dev`: product landing page, docs and open-core downloads (`hydrohq/hydrogo-web`).

## 5. Security & open-source readiness

- 3,000+ commit history scanned with Gitleaks; no true secrets found.
- `go run ./tools/check` runs build, vet, gofmt, migrations and the admin suite.
- Regional providers live in private modules; core has no hardcoded third-party business
  dependencies.
- Always call it "source-available" (PolyForm Shield), not "open source".

## 6. Development rules

1. **Performance first:** stdlib `net/http` over frameworks; reuse buffers via `sync.Pool`, avoid
   needless heap allocations in JSON serialization.
2. **Database integrity:** respect PostgreSQL RLS and SQLite WAL constraints; no unindexed scans on
   order or stock tables.
3. **Graceful failure:** shed excess load with 503 rather than growing memory.
4. **Clean boundaries:** commercial integrations plug in through the interfaces in
   `modules/commerce/` (`PaymentProvider`, `ShippingProvider`, `CarrierAdapter`).
