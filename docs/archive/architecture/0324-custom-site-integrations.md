# ADR-0324: Validated Custom Site Integrations

Status: Accepted

## Context

Operators need common analytics and support integrations without editing
generated layouts. Arbitrary script injection would bypass the platform's
content security policy and allow a site editor to execute code for visitors.

## Decision

Store snippets in the existing private, tenant-scoped site settings store. Only
active administrators may edit them (the current role model has no separate
superadmin or developer role). Bound each snippet to 16 KiB and reject inline
event handlers, dangerous embedding elements, non-HTTPS resources, and unknown
external hosts. The validator accepts a fixed allowlist for Google Tag Manager
and Analytics, Meta, Hotjar, and Zendesk. The server adds its per-response CSP
nonce to executable script and style elements and adds validated origins to the
matching CSP directives.

Apply snippets only to successful public HTML GET responses, with a bounded
response buffer. Exclude admin, API, authentication, media, static, and health
paths. Mark injected responses private and non-cacheable. Safe Mode suppresses
all snippets.

## Consequences

No schema migration or worker is required. Providers using hosts outside the
allowlist are rejected until an ADR-reviewed allowlist update is made. Dynamic
source construction inside custom JavaScript remains constrained by response
CSP.

## Compatibility and Operations

The existing private site-setting contract is shared by SQLite and
PostgreSQL. Disabling the Site module removes its management routes and runtime
provider. Rolling deployments remain compatible because the setting is
optional and unknown to earlier binaries.
