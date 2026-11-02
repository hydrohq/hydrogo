# Multi-tenancy

HydroGo can serve several separate customers/sites from one shared
database instead of one deployment per customer. This is a core,
opt-in architecture feature of the whole platform (every content
module, not something specific to news or commerce) -- it has existed
in the codebase for a while but had no written documentation until now.

## Default: single-tenant

With `tenancy.enabled: false` (the default), every row's internal
`tenant_id` column is just an empty string and every table behaves like
an ordinary single-customer database. You don't need to read the rest of
this page for a normal single-site deployment.

## Enabling it

```yaml
tenancy:
  enabled: true
  # true: an unrecognized Host header is refused (404) instead of falling
  # through to a default tenant. Set this once every real host you serve
  # is listed below.
  require_host: false
  hosts:
    customer-a.example.com: "tenant-a"
    customer-b.example.com: "tenant-b"
```

The tenant for a request is resolved **only** from this operator-configured
`hosts` map, matched against the request's `Host` header -- never from a
client-supplied header or query parameter, so a request can't claim a
tenant it doesn't belong to. See `internal/tenant.HostResolver`.

## What gets isolated

Every tenant-scoped table (`internal/tenant.BackfillTables` lists all of
them -- pages, news and its taxonomy, media, members, commerce, forms,
audit events, sessions, and more) carries a `tenant_id` column, and every
module's `repository/sqlstore` package threads the request's tenant
through every query (`RequireTenantScope` on the `Store`, the
`tenantFilter`/`s.p(n)` pattern you'll see repeated across
`modules/*/repository/sqlstore`). On PostgreSQL, every one of those
tables additionally gets row-level security
(`202609200003_postgres_tenant_rls` and `202611020002_postgres_rls_fail_closed`)
as a second, independent enforcement layer. RLS policies are strictly fail-closed:
`tenant_id = nullif(current_setting('app.current_tenant_id', true), '')`. If the
tenant setting is absent or empty, queries return zero rows and write checks fail.
Furthermore, the connection pool automatically executes `DISCARD TEMP; RESET ALL;`
upon releasing connections back to `pgxpool`, eliminating any risk of cross-transaction
tenant context leakage. SQLite and MySQL rely on application-level filtering alone
(there's no equivalent native RLS mechanism).

A table that's missing from both the per-module repository filtering and
the RLS table list is a real cross-tenant data leak, not just a style
inconsistency -- this is exactly what
[0024-news-taxonomy-tenant-isolation.md](decisions/0024-news-taxonomy-tenant-isolation.md)
found and fixed: `news_taxonomy_terms`/`news_taxonomy_relations` had been
added without either, so one tenant could read, overwrite or delete
another tenant's categories and tags. If you add a new tenant-scoped
table to any module, add it to `internal/tenant.BackfillTables`,
`tests/contract/postgres_rls_contract_test.go`'s `expectedRLSTables`, and
thread `tenant.Require(ctx)`/`RequireTenantScope` through its repository
the same way `modules/news/repository/sqlstore/taxonomy.go` now does --
in the same change, not as a follow-up.

## CLI

```bash
hydro tenant list [--json]                                    # configured tenants
hydro tenant validate                                          # config/schema sanity check
hydro tenant provision --tenant-id ID --admin-email E --admin-password-stdin
hydro tenant backfill (--tenant-id ID|--active) [--dry-run|--confirm]
```

`backfill` assigns pre-existing empty-tenant rows (e.g. after turning
`tenancy.enabled` on for a database that was single-tenant) to a real
tenant id; always run it with `--dry-run` first.

## Platform Orchestrator & Resource Governor

In addition to static `hosts` configuration and CLI provisioning, HydroGo includes a built-in **Platform Orchestrator** and **Resource Governor** (see [ADR 0029](decisions/0029-platform-orchestrator-and-resource-governor.md)):

- **Zero-Restart Provisioning & Domain Routing**: Workspaces provisioned via `POST /api/v1/admin/orchestrator/tenants` immediately register their `subdomain` and `custom_domain` in `HostResolver` and synchronize collection schemas without restarting the server.
- **Lifecycle & Subscription Auto-Degradation**: Each workspace has a `lifecycle_state` (`provisioning`, `active`, `readonly`, `suspended`, `archived`) and optional `trial_ends_at` / `subscription_ends_at` timestamps. When a trial or subscription expires, the workspace automatically degrades to `readonly` mode: public `GET`/`HEAD` requests continue serving the storefront while mutating requests return `402 Payment Required`.
- **Resource Governor**: Enforces per-tenant storage quotas (`max_storage_bytes`), collection record limits (`max_records`), token-bucket request rate limits (`rate_limit_rps`), in-flight concurrency semaphores (`max_concurrent_reqs`), upload size caps (`max_upload_bytes`), and monthly AI token quotas (`ai_monthly_token_limit`).
- **Caddy On-Demand TLS Gate**: `GET /api/v1/system/domain-check?domain=<host>` (loopback or `X-Orchestrator-Secret`) verifies whether a domain belongs to an active workspace before Caddy issues a TLS certificate.

## Multi-Adapter AI Copilot & Boutique Store Factory

- **Per-Workspace AI Adapters & BYOK**: Each workspace in `tenant_governance` can be assigned `gemini`, `openai`, `anthropic`, or `openai_compatible` (Ollama/Groq/DeepSeek) along with a model name, monthly token cap (`ai_monthly_token_limit`), and an optional AES-256-GCM encrypted API key (`ai_encrypted_api_key`).
- **Workspace Copilot & One-Click Undo**: Store owners can open the **AI Asistan** drawer in the admin header (`POST /api/v1/admin/ai/copilot`) to create/update/delete records or change storefront theme tokens in natural language, and revert any action in one click via `POST /api/v1/admin/ai/actions/{id}/undo`.
- **60-Second Boutique Factory (`tools/factory`)**:
  ```bash
  go run ./tools/factory \
    --tenant ipek-butik \
    --name "İpek Butik" \
    --phone "+905551112233" \
    --prompt "Kadın ipek gömlek ve keten takım satan lüks butik"
  ```
  Provisions the tenant workspace, applies `tenant_governance` quotas, syncs `content-types/boutique/*.yaml`, and seeds initial theme settings, categories, and products.
