# Admin Studio surface inventory — 2026-09-10

This inventory is based on reading the current Pages, News, and Commerce HTTP handlers for [Phase 0 of the Headless Admin Studio plan](../HEADLESS_ADMIN_STUDIO_PLAN.md). The goal is to separate metadata-generatable repetition from custom business rules; it is not a new API or runtime schema promise.

## Route ownership

| Domain | Admin owner | Machine API | Current integrated view |
|---|---|---|---|
| Pages | `/admin/pages/{id}` | `/api/pages/*` | `/public/pages/{locale}/{slug}/html` |
| News | `/admin/news/{id}` | `/api/news/*` | `/public/news/{locale}/{slug}/html` |
| Commerce | `/admin/commerce/*` | `/api/commerce/*` | `/shop/*` |

The `/public/*` routes remain for compatibility with the current integrated adapter. These are not extension route spaces for new headless clients; the target is the versioned `/api/v1/*` contract.

## Repeated Studio surface

| Capability | Pages | News | Commerce | Metadata generation status |
|---|---|---|---|---|
| Field label/type/required | Yes | Yes | Yes | `adminstudio.Collection` |
| Text, number, status, textarea widget | Partially | Yes | Yes | Shared form renderer |
| List columns and record action | Needed | Needed | Yes | Shared list renderer |
| CSRF field and HTML escaping | Yes | Yes | Yes | Shared form renderer |
| Optimistic version field | Yes | Yes | Module dependent | Handler/service responsibility |
| Role policy, audit, outbox | Yes | Yes | Yes | Not metadata; application service |

## Custom extension boundaries

### Pages

- Locale, slug, title, and body are standard form fields.
- The block editor (`hero`, `faq`, `cta`, `pricing_table`, `testimonials`), per-block payload validation, and media asset selection remain custom extensions.
- Draft/publish revision and public projection flow through `ContentService`. Studio can only share the form shell; it cannot push the publish operation directly to the DB.

### News

- Locale, slug, title, and body fit the metadata form. This screen has been migrated to the shared renderer.
- Schedule, publish, and retract state transitions use the generation/version contract in `news.Service`; they remain a separate Studio workflow action extension.

### Commerce

- Product catalog fields and the product list are rendered with shared metadata.
- Variant/stock, payment provider redirect, order/return, and refund financial side effects are not collection CRUD; they remain on the Commerce service and provider boundary.

## Next experiment gate

Before moving the standard fields of the Pages editor to the shared form, this interface must be proven for the block extension: form to block payload transfer, asset relation picker, CSRF/version forwarding, and invalid payload error presentation. Forcing blocks into a generic `relation` widget without this proof weakens ADR-0016's typed block contract.
