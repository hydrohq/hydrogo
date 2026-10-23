// Package transport exposes modules/commerce over HTTP: a public cart and
// checkout API (an anonymous visitor needs no account, see identity) and
// protected admin endpoints. Product/variant catalog reads go through the
// generic content-type public API (/api/v1/content/products), not here.
// Handlers only translate HTTP to Service calls; every rule lives in
// modules/commerce.
package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/notify"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// Route mirrors modules/news/transport.Route: Protected false is translated
// to router.Public centrally in cmd/hydro. Permission, when set, is the
// staff permission the route requires on top of being signed in.
type Route struct {
	Pattern    string
	Handler    http.Handler
	Protected  bool
	Permission string
}

// Staff permissions for the admin routes. They match internal/auth's
// PermissionCommerceRead/Write; commerce does not import internal/auth.
const (
	PermissionRead  = "admin.commerce.read"
	PermissionWrite = "admin.commerce.write"
)

const cartCookieName = "hydro_commerce_cart"

// notifyTimeout bounds each best-effort post-checkout call (receipt e-mail,
// operator chat ping) so a stalled mail server or chat API cannot hold the
// checkout response.
const notifyTimeout = 5 * time.Second

// Handler wires modules/commerce.Service to HTTP. MemberID is optional: a
// project that also enables modules/members supplies a resolver so a
// logged-in visitor's cart is tied to their account; commerce does not import
// modules/members.
type Handler struct {
	Service         commerce.Service
	Secure          bool
	DefaultCurrency string
	CartLimiter     security.RateLimiter
	CheckoutLimiter security.RateLimiter
	MemberID        func(*http.Request) (string, bool)
	// ClientIP returns the caller's address for rate limiting and for
	// providers that need it. It must only trust forwarding headers from
	// configured proxies; nil falls back to the connection's address.
	ClientIP func(*http.Request) string
	// PaymentProvider is optional: when set, a successful checkout also
	// starts a payment and the response says where to send the shopper. A
	// failed StartPayment never fails the checkout; the order already exists.
	PaymentProvider commerce.PaymentProvider
	// DisplayCurrencies/CurrencyRates back GET .../currencies. Display only:
	// checkout always charges DefaultCurrency.
	DisplayCurrencies []string
	CurrencyRates     map[string]float64
	// Mailer and Notifier are optional and best-effort.
	Mailer   OrderMailer
	Notifier notify.Sender
	// ShippingProvider is optional: when set, an admin can hand a paid order
	// to the carrier via POST .../orders/{id}/shipment.
	ShippingProvider commerce.ShippingProvider
}

// OrderMailer sends a customer-facing order receipt. mailer.NotificationSender
// satisfies it as-is.
type OrderMailer interface {
	SendOrderConfirmation(ctx context.Context, email, orderID, total string, items []mailer.OrderConfirmationItem) error
}

// Routes returns the public cart/checkout API and the protected admin API.
func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "GET /api/v1/commerce/cart", Handler: http.HandlerFunc(h.getCart)},
		{Pattern: "POST /api/v1/commerce/cart/items", Handler: http.HandlerFunc(h.addItem)},
		{Pattern: "DELETE /api/v1/commerce/cart/items/{id}", Handler: http.HandlerFunc(h.removeItem)},
		{Pattern: "POST /api/v1/commerce/checkout", Handler: http.HandlerFunc(h.checkout)},
		{Pattern: "GET /api/v1/commerce/stock/{variantID}", Handler: http.HandlerFunc(h.getStock)},
		{Pattern: "GET /api/v1/commerce/currencies", Handler: http.HandlerFunc(h.getCurrencies)},
		{Pattern: "GET /api/v1/commerce/checkout-config", Handler: http.HandlerFunc(h.getCheckoutConfig)},
		{Pattern: "GET /api/v1/commerce/orders/{id}", Handler: http.HandlerFunc(h.getOrder)},
		{Pattern: "POST /api/v1/commerce/orders/{id}/payment", Handler: http.HandlerFunc(h.startOrderPayment)},
		{Pattern: "POST /api/v1/admin/commerce/stock", Handler: http.HandlerFunc(h.setStock), Protected: true, Permission: PermissionWrite},
		{Pattern: "POST /api/v1/admin/commerce/discount-codes", Handler: http.HandlerFunc(h.createDiscountCode), Protected: true, Permission: PermissionWrite},
		{Pattern: "POST /api/v1/admin/commerce/orders/{id}/shipment", Handler: http.HandlerFunc(h.createShipment), Protected: true, Permission: PermissionWrite},
		{Pattern: "GET /api/v1/admin/commerce/orders", Handler: http.HandlerFunc(h.listOrdersAdmin), Protected: true, Permission: PermissionRead},
		{Pattern: "GET /api/v1/admin/commerce/orders/{id}", Handler: http.HandlerFunc(h.getOrder), Protected: true, Permission: PermissionRead},
	}
}

func (h Handler) defaultCurrency() string {
	if h.DefaultCurrency != "" {
		return h.DefaultCurrency
	}
	return "TRY"
}

func (h Handler) clientIP(r *http.Request) string {
	if h.ClientIP != nil {
		if ip := h.ClientIP(r); ip != "" {
			return ip
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// shopper resolves which cart the request belongs to: a logged-in member's,
// or an anonymous visitor's cookie, minting one on first use.
func (h Handler) shopper(w http.ResponseWriter, r *http.Request) (commerce.Shopper, error) {
	shopper := commerce.Shopper{Currency: h.defaultCurrency()}
	if h.MemberID != nil {
		if id, ok := h.MemberID(r); ok && id != "" {
			shopper.MemberID = id
			return shopper, nil
		}
	}
	if cookie, err := r.Cookie(cartCookieName); err == nil && cookie.Value != "" {
		shopper.AnonymousToken = cookie.Value
		return shopper, nil
	}
	token, err := randomToken()
	if err != nil {
		return commerce.Shopper{}, err
	}
	http.SetCookie(w, &http.Cookie{
		Name: cartCookieName, Value: token, Path: "/", MaxAge: int((30 * 24 * time.Hour).Seconds()),
		HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode,
	}) // #nosec G124 -- Secure follows the secure_cookies setting (off only for plain-HTTP development); the cookie is HttpOnly and SameSite
	shopper.AnonymousToken = token
	return shopper, nil
}

func randomToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func rateLimited(w http.ResponseWriter, limiter security.RateLimiter, key string) bool {
	if limiter == nil || limiter.Allow(key) {
		return false
	}
	security.WriteRateLimitedResponse(w, "too many requests", time.Minute)
	return true
}

// writeServiceError maps commerce's sentinel errors to fixed client
// messages. Anything unrecognised is a 500 whose detail stays in the log
// (security.ErrorMiddleware never echoes 5xx text).
func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, commerce.ErrOrderNotFound):
		api.WriteError(w, r, http.StatusNotFound, "order not found", nil)
	case errors.Is(err, commerce.ErrOutOfStock):
		api.WriteError(w, r, http.StatusConflict, "one or more items in your cart are no longer available in the requested quantity", nil)
	case errors.Is(err, commerce.ErrDiscountCodeInvalid):
		api.WriteError(w, r, http.StatusUnprocessableEntity, "this discount code cannot be applied to your order", map[string]string{"discount_code": "invalid"})
	case errors.Is(err, commerce.ErrIdempotencyConflict):
		api.WriteError(w, r, http.StatusConflict, "this idempotency key was already used for a different checkout", nil)
	case errors.Is(err, commerce.ErrCartChanged):
		api.WriteError(w, r, http.StatusConflict, "the cart changed during checkout; please try again", nil)
	case errors.Is(err, commerce.ErrEmptyCart):
		api.WriteError(w, r, http.StatusUnprocessableEntity, "cart is empty", nil)
	case errors.Is(err, commerce.ErrUnknownVariant):
		api.WriteError(w, r, http.StatusUnprocessableEntity, "this item is not for sale", map[string]string{"variant_id": "unknown"})
	case errors.Is(err, commerce.ErrCurrencyMismatch):
		api.WriteError(w, r, http.StatusUnprocessableEntity, "this item is priced in a different currency than the cart", nil)
	case errors.Is(err, commerce.ErrNoPriceSource):
		api.WriteError(w, r, http.StatusServiceUnavailable, "the store has no price source configured", nil)
	case errors.Is(err, commerce.ErrOrderNotPayable):
		api.WriteError(w, r, http.StatusConflict, "this order is no longer awaiting payment", nil)
	case errors.Is(err, commerce.ErrOrderNotShippable):
		api.WriteError(w, r, http.StatusConflict, "only paid, unshipped orders can be handed to a carrier", nil)
	default:
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
	}
}

// getCurrencies is a public config echo for a storefront's currency
// switcher.
func (h Handler) getCurrencies(w http.ResponseWriter, r *http.Request) {
	api.WriteData(w, http.StatusOK, map[string]any{
		"default": h.defaultCurrency(),
		"display": h.DisplayCurrencies,
		"rates":   h.CurrencyRates,
	}, nil)
}

// getCheckoutConfig is a public config echo so a storefront can estimate a
// total; the checkout response is authoritative.
func (h Handler) getCheckoutConfig(w http.ResponseWriter, r *http.Request) {
	pricing := h.Service.Pricing
	api.WriteData(w, http.StatusOK, map[string]any{
		"tax_rate":                  pricing.TaxRate,
		"tax_inclusive":             pricing.TaxInclusive,
		"shipping_flat_rate_cents":  pricing.ShippingFlatRateCents,
		"shipping_free_above_cents": pricing.ShippingFreeAboveCents,
	}, nil)
}
