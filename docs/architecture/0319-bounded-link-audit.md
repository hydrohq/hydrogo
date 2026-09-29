# ADR-0319: Bounded public link and asset audit

- **Status:** Accepted for operator-initiated scans
- **Date:** 2026-09-29
- **Related decisions:** [ADR-0024](0024-optional-redis-rate-limiting-and-queue-boundary.md), [ADR-0030](0030-client-address-trust-boundary.md), [ADR-0268](0268-seo-performance-and-discovery-runtime.md)

## Decision

`hydro scan-links` and the authenticated Site audit page use a shared pure-Go crawler. A scan starts at the configured public origin, follows same-origin pages, and checks external anchors and in-page image/script/frame assets. Findings report the source page, parsed page title, failed URL, status or unreachable state, visible anchor/alt text, and a status-specific suggested action. The CLI emits CSV. The Admin Studio starts scans in a bounded process-local manager and shows the latest completed result; a report with findings adds a dashboard warning. A non-CGI server schedules a monthly scan at 03:00 UTC on the first day of each month for configured tenants.

The crawler uses a maximum of 500 pages, 5,000 checked targets, 1,000 findings, 16 concurrent workers, and 2 MiB of HTML per page. The configured request delay defaults to 100 ms. It retries one transient network, 429, or 5xx failure before recording it. Its default dialer rejects private, loopback, link-local, carrier-grade NAT and benchmarking networks to prevent a public page from turning an audit into a request to local services. An operator may explicitly allow private targets for a trusted local CLI scan. HTTP redirects are capped at five. Admin scans are tenant-keyed, limited to one running scan per tenant, and reports expire after 35 days.

The Site module owns the routes and manager; inactive Site editions create no scanner or worker. Scans are read-only and require no migration, distributed cache, or persistent job. Restarting a process stops its in-memory scans and drops their reports. Monthly timer coordination is process-local, so multiple active server instances can repeat a scheduled scan; deployments that require singleton scheduling must coordinate at the deployment layer. CGI mode does not run the monthly schedule.

## Verification

Tests cover same-origin traversal, external links, image assets, page titles, anchor text, suggested actions, transient retries, CSV output, monthly schedule timing, tenant isolation, URL validation, and private-network blocking. Full repository tests remain the integration gate. Production crawl volume and domain-specific rate limits require workload tuning.
