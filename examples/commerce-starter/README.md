# HydroGo commerce starter

A server-rendered storefront (SvelteKit, `adapter-node`): a product catalog,
cart, guest checkout and PayTR payment against a HydroGo backend with the
`commerce` preset and the `commerce` module enabled. No build step is needed
when content changes -- every request renders from the live API.

## Run it

1. Start a HydroGo project with the catalog content types
   (`content-types/commerce/*.yaml`) and the `commerce` module enabled, with
   a `commerce.paytr` section in its config if you want real payments (see
   the root `docs/decisions/0014-commerce-paytr-provider.md`). Point
   `commerce.paytr.ok_url`/`fail_url` at this site's `/odeme/sonuc`.
2. `cp .env.example .env` and fill in `HYDRO_API_ORIGIN`.
3. `npm install && npm run dev`.

In production (`npm run build && npm start`, or the Dockerfile),
`@sveltejs/adapter-node` needs an `ORIGIN` environment variable set to this
site's own public URL -- its CSRF protection compares every write's
`Origin` header against it, and without it, form submissions behind a
reverse proxy can be refused.

## Pages

`/` (per-category sections), `/kategori/[slug]` (category landing, with a
sort switcher), `/arama?q=` (search), `/urun/[slug]` (product detail:
gallery, variant image swap, related products), `/sepet`, `/odeme` (with a
discount code field and a tax/shipping estimate), `/siparis/[id]` (full
subtotal/discount/tax/shipping/total receipt), `/odeme/sonuc` (PayTR
redirect landing), `/para-birimi/[code]` (switches the display currency).

## Tax, shipping, discount codes, currencies

All four are configured on the HydroGo side (`commerce.tax`,
`commerce.shipping`, `commerce.currencies`; discount codes via `POST
/api/v1/admin/commerce/discount-codes`), not in this starter -- see
`docs/decisions/0017-commerce-tax-shipping-discounts-currencies.md`. The
currency switcher only changes how a price is *shown*
(`convertDisplayPrice`, a static operator-set rate); what actually gets
charged is always `commerce.currencies.default`, never the display
currency.

## How it talks to HydroGo

- The product catalog (`src/lib/server/hydro.ts`, a trimmed copy of
  `packages/client`) reads the generic content-type API
  (`/api/v1/content/products` etc.) straight from each page's server `load`
  function -- no client-side fetch, no build-time fetch.
- Cart, checkout and payment (`src/lib/server/commerce.ts`) go through
  `modules/commerce`'s own API. There is no privileged secret to protect
  here (contrast with `examples/crm-starter`'s staff session, which keeps
  HydroGo's cookie server-side): a visitor's cart cookie is proxied straight
  through, verbatim, between the browser and HydroGo, with this SvelteKit
  server only in the middle to attach the CSRF header every write needs.
- `/odeme/sonuc` is where PayTR's `ok_url`/`fail_url` redirect lands. Per
  PayTR's own docs that redirect carries no reliable payment status, so the
  page does nothing but forward to `/siparis/<id>`, which asks HydroGo for
  the real, server-verified order status.

## What this is not

Not a production storefront: no real multi-currency settlement (display
only, see above), no member accounts, no inventory beyond what
`modules/commerce` already tracks, no drag-and-drop
theme editor (and none is planned for this starter specifically -- if
HydroGo ever gets a visual page editor it belongs at the project level,
see `docs/decisions/0016-commerce-starter-catalog-sections.md`). It is
a starting point, designed for this module rather than copied from
`examples/sveltekit-starter` or `examples/hydropost-news-starter` (see the
root `AGENTS.md`).
