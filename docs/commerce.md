# Commerce

The `commerce` module adds a cart, guest/member checkout, order lifecycle,
payment and shipping provider plumbing, tax/discounts and operator tooling
on top of a catalog that lives in ordinary content types. It is opt-in: add
`commerce` to `modules:` in your config (or start from `hydro new --preset
commerce`).

## Catalog

Products, variants and categories are content types, not module code --
`content-types/commerce/{categories,products,variants,product_images}.yaml`.
They get the normal content-type treatment: admin screens, search/filter,
and a public read API at `/api/v1/content/{categories,products,variants}`.
Stock, pricing at checkout time, carts and orders are the commerce module's
own SQL tables; see `docs/decisions/0012-commerce-catalog-as-content-types.md`
for why the split sits there.

## Public API

| Route | Notes |
|---|---|
| `GET /api/v1/commerce/cart` | The caller's cart (member session or anonymous cart cookie). |
| `POST /api/v1/commerce/cart/items` | Add a line item; price is always looked up server-side from the catalog, a client-sent price is ignored. |
| `DELETE /api/v1/commerce/cart/items/{id}` | Remove a line item. |
| `GET /api/v1/commerce/stock/{variantID}` | Current available/reserved stock for one variant. |
| `GET /api/v1/commerce/currencies` | Configured default/display currencies and rates (presentation only). |
| `GET /api/v1/commerce/checkout-config` | What checkout needs from the client: tax rate/inclusivity, shipping fee/threshold, enabled payment providers. |
| `POST /api/v1/commerce/checkout` | Places the order from the current cart. Idempotent per `(member or anonymous cart, idempotency_key)` -- a retried request with the same key returns the same order instead of creating a second one. |
| `GET /api/v1/commerce/orders/{id}` | One order (its own cart/member only). |
| `POST /api/v1/commerce/orders/{id}/payment` | Starts a payment with the configured provider (if any) and returns its redirect/token. |

All public routes are rate-limited; writes go through the same CSRF/session
rules as the rest of the API.

## Admin API

Requires `admin.commerce.read` or `admin.commerce.write` (checked through
the shared permission middleware, like every other admin route):

| Route | Permission |
|---|---|
| `GET /api/v1/admin/commerce/orders` | read |
| `GET /api/v1/admin/commerce/orders/{id}` | read |
| `POST /api/v1/admin/commerce/stock` | write |
| `POST /api/v1/admin/commerce/discount-codes` | write |
| `POST /api/v1/admin/commerce/orders/{id}/shipment` | write |

The admin Orders screen (`/admin/commerce`, shown only when the `commerce`
module is on) lists orders with a payment-status filter and search, and a
detail view with customer, address, line items, pricing breakdown and a
"hand to carrier" action for paid, unshipped orders.

## CLI

```bash
hydro commerce reconcile [--provider NAME] [--hold 30m] [--stale-after 15m] [--limit 100] [--json]
```

A read-only consistency sweep for stale pending orders, a payment
notification that arrived but never flipped the order's status, and
stock-reservation drift. The same sweep also runs in the background every
five minutes and logs a warning when a report is not clean. See
`docs/decisions/0018-commerce-reconciliation.md`.

## Provider Interfaces & Decoupled Architecture

In accordance with core architectural guidelines (no customer-specific or domestic courier/gateway implementations in core binary), payment and shipping carriers are decoupled into clean Go interfaces:

### `PaymentProvider`
```go
type PaymentProvider interface {
    StartPayment(ctx context.Context, order Order, items []OrderItem) (PaymentStart, error)
}
```
Extended lifecycles (authorize, capture, refund) can optionally implement `AdvancedPaymentProvider`.

### `ShippingProvider`
```go
type ShippingProvider interface {
    Name() string
    CreateShipment(ctx context.Context, order Order, items []OrderItem) (ShipmentResult, error)
}
```
Carriers supporting tracking and cancellation can optionally implement `CarrierAdapter`.

### Registering External Providers
Operators can register custom provider implementations during application startup or in starter packages:
```go
import (
    "github.com/ardaozyildirim/hydrogo/cmd/hydro"
    "github.com/ardaozyildirim/hydrogo/examples/commerce-starter/extensions/yurticikargo"
)

func init() {
    hydro.RegisterShippingProvider(yurticikargo.Provider{
        Config: yurticikargo.Config{
            WSUserName: os.Getenv("YURTICI_USER"),
            WSPassword: os.Getenv("YURTICI_PASS"),
        },
    })
}
```

## Configuration

See the `commerce:` section of `config.example.yaml` for optional keys (tax, shipping fees, currencies) and `notifications.telegram` for order/reconciliation alerts. Everything under `commerce:` is optional -- with it empty, cart and checkout still work with no payment provider, no carrier, no tax, no shipping fee and TRY/USD/EUR as the configured currency.

Specific regional integrations (such as Yurtiçi Kargo) reside in `examples/commerce-starter/extensions/yurticikargo` as decoupled extensions.

## Starter

`examples/commerce-starter` is a SvelteKit storefront (catalog, cart, guest and member checkout, order confirmation, member login/account/order history) built against this API. `hydro new --starter commerce` copies it into a project's `frontend/`.
