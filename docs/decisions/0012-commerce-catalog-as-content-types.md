# 0012: Commerce catalog as content types, not a new engine feature

- Status: Accepted (Phase 1 of 4); the remaining phases (cart/checkout module,
  PayTR provider, storefront starter) are proposed, not yet built.
- Date: 2026-10-23

## Context

HydroGo removed its commerce module in `ac1de094` (ADR `0001-cms-focus.md`) as
one casualty among many in a broader six-week sprawl cleanup — not because
commerce itself was judged a bad fit. The project now wants e-commerce back,
reusable across projects (starting with a leather-goods storefront) instead
of maintaining a separate Medusa.js deployment per client. Large parts of the
old module's plumbing are still wired in (permissions, CORS allowlist, a
tenant-backfill list, two contract tests) but blocked by one explicit guard
in `internal/config/validate.go`; its migrations are still on disk per rule
3 (never deleted) and still checksummed into `migrations/baseline/manifest.yml`.

The question this record answers: does product/variant/category *catalog*
data need new core engine capability, or does the existing declarative
content-type system already cover it? `content-types/erp/products.yaml`
(an existing preset) already demonstrated most of the needed shape —
sequence numbers, rollups, computed totals, guarded state machines — just
not applied to a public storefront catalog yet.

## Decision

Catalog data (`categories`, `products`, `variants`) is three plain content
types under `content-types/commerce/`, registered as a new `commerce` preset
in `internal/scaffold.go` (`hydro new --preset commerce`) — no new engine
feature, no Go code. Confirmed this is sufficient, not merely assumed:

- **Relations model variants with no special "variant" feature.** `variants.yaml`
  is just `{kind: relation, to: products, required: true}` plus its own
  `sku`/`size`/`color`/`price_override` fields — the generic relation
  mechanism `content-types/erp/order_lines.yaml` already uses for order
  lines.
- **Stock is deliberately NOT a field on `variants`.** Selling the last unit
  safely needs an atomic, conditional SQL update; the generic collections
  store (`internal/collections/sqlstore`) only has optimistic version-locking,
  confirmed by reading `store.go` — no compare-and-swap primitive exists
  there. Stock belongs in the Phase 2 `modules/commerce` module's own table,
  keyed loosely by a variant's content-type id (no DB foreign key, since
  content records live in the generic `hydro_collection_records` table).
- **Images are a single `cover_image` media field, not a gallery.** `many:`
  only validates on `relation` fields per `internal/contenttypes/definition.go`'s
  `Validate()` — a multi-image gallery would need a separate relation-based
  content type (the existing `galleries.yaml` pattern), deferred until a real
  project asks for it rather than built speculatively.

## Verification

Beyond `internal/scaffold`'s existing `TestEveryPresetScaffoldsAValidProject`
(now covers `commerce` automatically, confirms the YAML loads and the admin
collection registry accepts it), this was run against a real `hydro serve`
instance from a scaffolded `--preset commerce` project: created a category,
a product (with a real uploaded media asset as `cover_image` and a relation
to the category), and a variant (relation to the product) through the real
admin API; published the product through the real `POST .../workflow`
endpoint; then confirmed, against the live public API with no custom code
anywhere: `GET /api/v1/content/products` lists it, `?search=kemer` finds it
by name, `?filter[category]=<id>` and `?filter[product]=<id>` correctly
scope by relation, and `GET /api/v1/admin/schema` lists all three
collections for the admin SPA to render generically.

## Consequences

- A storefront catalog is available today with zero Go/Svelte code, exactly
  matching the "declare it once" model the rest of HydroGo follows.
- The `commerce` preset currently only enables `[core, auth, media, collections]`
  — no `commerce` module yet, since it doesn't exist. Phase 2 extends this
  preset's module list once cart/checkout/payments land; this preset name is
  not a placeholder that will be renamed later.
- Risk: none of the "already-wired-but-blocked" old commerce plumbing
  (permissions, CORS allowlist, tenant backfill, contract tests) was touched
  in this phase — it still assumes the old `commerce_products`/`commerce_variants`
  SQL tables, which this phase deliberately does not create. Phase 2 must
  update those references to the new table names, not just unblock them.
