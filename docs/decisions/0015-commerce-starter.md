# 0015: `examples/commerce-starter`

- Status: Accepted
- Date: 2026-10-23

## Context

[0013](0013-commerce-cart-checkout-module.md) and
[0014](0014-commerce-paytr-provider.md) built a complete commerce API with
no public-facing site to use it from. AGENTS.md is explicit that HydroGo
does not render a public website itself; a starting point for one belongs
in `examples/`, built from scratch rather than copied from
`examples/sveltekit-starter` or `examples/hydropost-news-starter`.

The one real design question was whether the cart cookie needs
`examples/crm-starter`'s server-side session-store indirection (the
browser only holds an opaque panel session id; HydroGo's own staff
session/CSRF cookie never reaches it). It doesn't: a staff session is a
privileged credential worth hiding from the browser's own JS context; an
anonymous shopper's cart cookie is not a secret held on their behalf, it
*is* their identity, no different in kind from a session id any other
e-commerce site hands the browser directly.

## Decision

A server-rendered SvelteKit storefront (`adapter-node`, matching
`examples/sveltekit-starter`'s "every request renders live" choice,
AGENTS.md rule 12): product grid and detail pages read the catalog content
types straight from each page's server `load` (a trimmed copy of
`packages/client`'s generic content client, `src/lib/hydrogo.ts` — no
commerce-specific additions to `@hydrogo/client` itself, since cart/
checkout need cookie forwarding that client was never built for).

`src/lib/server/commerce.ts` proxies `modules/commerce`'s cookies (cart,
CSRF, member session — exactly, by name) between the browser and HydroGo,
unchanged: it attaches `X-CSRF-Token` from the forwarded `hydro_csrf`
cookie on every write and replays any `Set-Cookie` HydroGo returns as this
site's own cookie of the same name. There is no session store because there
is no secret to keep off the browser — see Context above. The root layout's
server load always calls `GET /api/v1/commerce/cart` once, which both shows
the cart count in the header and warms the CSRF cookie before any page's
form action tries its first write (a cold POST with no prior GET gets a
403 in the same response that mints the cookie it needed — see
`internal/security/csrf.go` — so every page that could lead to a mutation
needs that warm-up first).

New, small additions to `modules/commerce` this storefront needed and found
missing by actually being built against the real API, not assumed:
- `GET /api/v1/commerce/stock/{variantID}` (public): a product page needs
  to know if a variant is sold out; the admin-only `reserved` count is
  checkout bookkeeping, not a shopper fact, so this returns only
  `available`. A never-stocked variant reads as zero, not an error.
- `GET /api/v1/commerce/orders/{id}` (public): a guest has no account to
  log into for order history, so the order confirmation page needs a read
  scoped only by the order's own unguessable id — the same trust model
  `POST .../orders/{id}/payment` ([0014](0014-commerce-paytr-provider.md))
  already relies on.
- `paytr.Provider.StartPayment` now appends `?oid=<id>` to the configured
  `OKURL`/`FailURL`: PayTR's redirect carries no reliable payment status
  (see 0014), so the only thing worth passing through it is which order to
  ask the real, server-verified status about. `/odeme/sonuc` does nothing
  but read that id and forward to `/siparis/<id>`.
- `internal/security/cors.go`'s public-read allowlist gained these two new
  GETs (not cart, which is cookie-identity-scoped and would not survive
  cross-origin `SameSite=Lax` anyway) for a storefront hosted on its own
  domain.

`internal/scaffold.Starters()` gained `"commerce"`
(`hydro new --starter commerce`).

## Consequences

- Verified against a real `hydro serve` instance end to end, not mocked: a
  real catalog (category, product with an uploaded cover image, variant)
  created and published through the real admin API, then the built,
  production (`node build`) storefront driven through the full guest
  flow — browse, add to cart, checkout, a forged-but-correctly-signed PayTR
  callback (signed independently in Python, matching the discipline from
  0014) — confirmed the order flips from "Ödeme bekleniyor" to "Ödendi" on
  the confirmation page, stock settles at the correct `available`/`reserved`
  split, and the sold-out state renders correctly once stock reaches zero.
- Found and fixed live: the idempotency-key cookie (`checkout_idem`, local
  to this starter, not one of HydroGo's own proxied cookies) was marked
  `Secure` by SvelteKit's own default heuristic even with
  `PUBLIC_SECURE_COOKIES=false`, which a plain-HTTP browser silently
  refuses to store at all — breaking the exact "retry reuses the same
  order" guarantee the cookie exists for. Fixed by setting `secure`
  explicitly from the same config everywhere this starter sets a cookie,
  rather than trusting the framework's default. Exactly the kind of gap
  only a real request against a real server surfaces.
- `@sveltejs/adapter-node` deployments of this starter need `ORIGIN` set
  (its own CSRF protection compares the `Origin` header against it); the
  README and `.env.example` call this out.
