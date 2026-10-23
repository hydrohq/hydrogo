# 0016: Commerce starter catalog sections (category pages, search, galleries)

- Status: Accepted
- Date: 2026-10-23

## Context

[0015](0015-commerce-starter.md) shipped a working storefront, but a flat
product grid on one page is thinner than what the business actually wants
to launch with: category landing pages, a way to find a product by name,
and a product page that shows more than one photo. The alternative —a
drag-and-drop visual theme editor, Shopify-style— was discussed and
explicitly rejected for this starter: it is a different kind of product
(a page-builder SaaS), it contradicts AGENTS.md's "not a framework or
product factory," and if HydroGo ever gets a visual editor it belongs at
the project level, not bolted onto the commerce starter specifically. What
this starter needed instead was more pre-built, swappable pieces that a
per-brand build (by a person or an agent editing the code directly) can
mix and match quickly — closer to "more starter content" than "more
product."

## Decision

Two small, additive content-type changes (no migration needed: content
types are generic `data_json` records, not typed SQL columns, so a new
field or a new type is just a schema change the next request picks up):
- `variants.yaml` gained `image` (media, optional): a colour variant can
  show its own photo; falls back to the product's `cover_image` when
  empty.
- New `product_images.yaml` (product relation, image, sort_order): a
  product's extra gallery photos beyond its cover image, as its own child
  type, the same pattern `variants` already uses to relate to `products`.

Storefront additions, all reading the existing public content/commerce
APIs with no further backend changes:
- `/kategori/[slug]`: a real collection landing page (name, description,
  products, a sort switcher — newest/price/name, passed straight through
  as the content API's own `sort` parameter). The header's category nav
  now links here instead of the old `/?kategori=` homepage filter, which
  is removed.
- `/arama`: a search box in the header posts to a results page using the
  content API's existing `search` parameter (already enabled on
  `products.name`/`description`/`sku` by [0012](0012-commerce-catalog-as-content-types.md)'s
  `searchable: true` flags) — no new backend capability, just a page that
  uses one that already existed.
- The homepage becomes up to several "by category" sections (a few
  products each, a "tümünü gör" link to the category page) instead of one
  flat grid; an empty or just-launched catalog with no categorised
  products falls back to the original flat "newest" grid so it never shows
  nothing.
- The product page gained a gallery (cover image + `product_images`,
  clickable thumbnails) that swaps to a variant's own `image` when one is
  selected, and a "benzer ürünler" (related products) strip from the same
  category, excluding the current product.
- `ProductCard`/`ProductGrid` were factored out of the homepage into
  `src/lib/components/`, now shared by the homepage, category, search and
  related-products views instead of four copies of the same markup.

## Consequences

- `toHttpError` (the shared catalog-load error wrapper) had to learn to
  let a load function's own `error(...)`/`redirect(...)` pass through
  unchanged (`isHttpError`/`isRedirect`) instead of catching and rewriting
  every exception thrown inside its try block as a 502 indiscriminately.
  Found by the automated smoke test, not by inspection: an unknown
  product's intended 404 was coming back as a 502 before this fix.
- Verified against a real `hydro serve` instance: two categories, two
  products (one with a second, variant-specific photo and a gallery
  image), confirmed live — the homepage's per-category sections, the
  category page's sort links (verified the content API itself accepts
  `sort=-base_price` without error), search finding a real match and
  correctly showing nothing for a non-match, the product page's gallery
  thumbnails and variant-driven image swap, and "benzer ürünler" showing
  the other product in the same category while excluding itself.
- `npm run smoke`'s stub-API run extended to cover the new routes
  (`/kategori/does-not-exist` is a 404, `/arama` renders with no results)
  without needing a real backend.
