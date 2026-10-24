// Package transport exposes modules/commerce over HTTP: a public cart and
// checkout API (an anonymous visitor needs no account, see cookie.go) and a
// small protected stock-management endpoint for admins. Product/variant
// catalog reads go through the generic content-type public API
// (/api/v1/content/products) already, not through here.
package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	"github.com/ardaozyildirim/hydrogo/internal/notify"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// Route mirrors modules/news/transport.Route: Protected false is translated
// to router.Public centrally in cmd/hydro/routes.go, the same way every
// other module's routes are.
type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

const cartCookieName = "hydro_commerce_cart"

// Handler wires modules/commerce.Service to HTTP. MemberID is optional: a
// project that also enables modules/members can supply a resolver so a
// logged-in visitor's cart is tied to their account instead of an anonymous
// cookie; commerce does not import modules/members directly (no module
// dependency edge needed for that), so the resolver is injected by
// cmd/hydro at wiring time.
type Handler struct {
	Service            commerce.Service
	DB                 database.Executor
	Secure             bool
	DefaultCurrency    string
	CartLimiter        security.RateLimiter
	CheckoutLimiter    security.RateLimiter
	RequireTenantScope bool
	MemberID           func(*http.Request) (string, bool)
	// PaymentProvider is optional: when set, a successful checkout also
	// starts a payment with it (paytr.Provider, currently the only one) and
	// the response carries wherever it says to send the shopper. A failed
	// StartPayment never fails the checkout itself -- the order already
	// exists -- the response simply has no payment field, matching the
	// generic-add-to-cart-then-pay flow.
	PaymentProvider commerce.PaymentProvider
	// DisplayCurrencies/CurrencyRates back GET .../currencies, a
	// storefront's currency switcher. Display-only: changing them never
	// changes what checkout actually charges (always DefaultCurrency) --
	// see config.Currencies' own doc comment for why.
	DisplayCurrencies []string
	CurrencyRates     map[string]float64
	// Mailer and Notifier are both optional and both best-effort: a
	// checkout that already succeeded must never fail because an email or
	// a chat ping did. OrderMailer is satisfied by
	// mailer.NotificationSender directly (same method shape), no adapter
	// needed.
	Mailer   OrderMailer
	Notifier notify.Sender
	// ShippingProvider is optional: when set, an admin can hand a paid
	// order to it via POST .../orders/{id}/shipment (Yurtiçi Kargo
	// currently; see modules/commerce.ShippingProvider). A project with
	// none configured still gets full cart/checkout/payment, just no
	// carrier integration -- same shape as PaymentProvider above.
	ShippingProvider commerce.ShippingProvider
}

// OrderMailer sends a customer-facing order receipt. Defined here, not in
// internal/mailer, because the order/currency formatting is commerce's
// concern -- mailer.NotificationSender satisfies this interface as-is.
type OrderMailer interface {
	SendOrderConfirmation(ctx context.Context, email, orderID, total string, items []mailer.OrderConfirmationItem) error
}

func (h Handler) defaultCurrency() string {
	if h.DefaultCurrency != "" {
		return h.DefaultCurrency
	}
	return "TRY"
}

func (h Handler) memberID(r *http.Request) string {
	if h.MemberID == nil {
		return ""
	}
	id, ok := h.MemberID(r)
	if !ok {
		return ""
	}
	return id
}

// identity resolves which cart the request belongs to: a logged-in
// member's, or an anonymous visitor's cookie -- minting and setting a new
// one if this is their first cart action. Exactly one of the two return
// values is non-empty, matching commerce.Cart's own invariant.
func (h Handler) identity(w http.ResponseWriter, r *http.Request) (memberID, anonymousToken string, err error) {
	if id := h.memberID(r); id != "" {
		return id, "", nil
	}
	if cookie, cookieErr := r.Cookie(cartCookieName); cookieErr == nil && cookie.Value != "" {
		return "", cookie.Value, nil
	}
	token, err := randomToken()
	if err != nil {
		return "", "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name: cartCookieName, Value: token, Path: "/", MaxAge: int((30 * 24 * time.Hour).Seconds()),
		HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode,
	}) // #nosec G124 -- Secure follows the secure_cookies setting (off only for plain-HTTP development); the cookie is HttpOnly and SameSite
	return "", token, nil
}

func randomToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func rateLimited(w http.ResponseWriter, limiter security.RateLimiter, key string) bool {
	if limiter == nil {
		return false
	}
	if limiter.Allow(key) {
		return false
	}
	security.WriteRateLimitedResponse(w, "too many requests", time.Minute)
	return true
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return r.RemoteAddr
}

// Routes returns the public cart/checkout API and the protected stock
// endpoint.
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
		{Pattern: "POST /api/v1/admin/commerce/stock", Handler: http.HandlerFunc(h.setStock), Protected: true},
		{Pattern: "POST /api/v1/admin/commerce/discount-codes", Handler: http.HandlerFunc(h.createDiscountCode), Protected: true},
		{Pattern: "POST /api/v1/admin/commerce/orders/{id}/shipment", Handler: http.HandlerFunc(h.createShipment), Protected: true},
	}
}

type cartResponse struct {
	ID    string         `json:"id"`
	Items []cartItemView `json:"items"`
}

type cartItemView struct {
	ID             string `json:"id"`
	VariantID      string `json:"variant_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
}

func (h Handler) getCart(w http.ResponseWriter, r *http.Request) {
	memberID, anonymousToken, err := h.identity(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	cart, err := h.Service.GetOrCreateCart(r.Context(), h.DB, memberID, anonymousToken, h.defaultCurrency())
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	writeCart(w, r, h, cart)
}

type addItemRequest struct {
	VariantID      string `json:"variant_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
}

func (h Handler) addItem(w http.ResponseWriter, r *http.Request) {
	memberID, anonymousToken, err := h.identity(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	if rateLimited(w, h.CartLimiter, memberID+anonymousToken+clientIP(r)) {
		return
	}
	var input addItemRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	if strings.TrimSpace(input.VariantID) == "" || input.Quantity <= 0 || input.UnitPriceCents < 0 {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "variant_id, a positive quantity and a non-negative unit_price_cents are required", nil)
		return
	}
	cart, err := h.Service.GetOrCreateCart(r.Context(), h.DB, memberID, anonymousToken, h.defaultCurrency())
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	if err := h.Service.AddItem(r.Context(), h.DB, cart.ID, input.VariantID, input.Quantity, input.UnitPriceCents); err != nil {
		api.WriteError(w, r, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}
	writeCart(w, r, h, cart)
}

func (h Handler) removeItem(w http.ResponseWriter, r *http.Request) {
	memberID, anonymousToken, err := h.identity(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	cart, err := h.Service.GetOrCreateCart(r.Context(), h.DB, memberID, anonymousToken, h.defaultCurrency())
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	if err := h.Service.Repository.RemoveItem(r.Context(), h.DB, cart.ID, r.PathValue("id")); err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	writeCart(w, r, h, cart)
}

func writeCart(w http.ResponseWriter, r *http.Request, h Handler, cart commerce.Cart) {
	items, err := h.Service.Repository.ListItems(r.Context(), h.DB, cart.ID)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	views := make([]cartItemView, 0, len(items))
	for _, item := range items {
		views = append(views, cartItemView{ID: item.ID, VariantID: item.VariantID, Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents})
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, cartResponse{ID: cart.ID, Items: views}, nil)
}

type checkoutRequest struct {
	GuestName       string `json:"guest_name"`
	GuestEmail      string `json:"guest_email"`
	GuestPhone      string `json:"guest_phone"`
	GuestAddress    string `json:"guest_address"`
	GuestCity       string `json:"guest_city"`
	GuestCountry    string `json:"guest_country"`
	GuestPostalCode string `json:"guest_postal_code"`
	IdempotencyKey  string `json:"idempotency_key"`
	DiscountCode    string `json:"discount_code"`
	// ItemTitles/ItemSKUs are keyed by variant_id: cart items only carry
	// variant_id/quantity/price (commerce has no access to the
	// content-type catalog records), so the caller -- the storefront,
	// which already resolved these for its own cart display -- supplies
	// them here for the order receipt.
	ItemTitles map[string]string `json:"item_titles"`
	ItemSKUs   map[string]string `json:"item_skus"`
}

func (h Handler) checkout(w http.ResponseWriter, r *http.Request) {
	memberID, anonymousToken, err := h.identity(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	if rateLimited(w, h.CheckoutLimiter, memberID+anonymousToken+clientIP(r)) {
		return
	}
	var input checkoutRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "idempotency_key is required", nil)
		return
	}
	if memberID == "" && strings.TrimSpace(input.GuestEmail) == "" {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "guest_email is required for a guest checkout", nil)
		return
	}
	cart, err := h.Service.GetOrCreateCart(r.Context(), h.DB, memberID, anonymousToken, h.defaultCurrency())
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	order, err := h.Service.Checkout(r.Context(), commerce.CheckoutRequest{
		CartID: cart.ID, MemberID: memberID,
		GuestName: input.GuestName, GuestEmail: input.GuestEmail, GuestPhone: input.GuestPhone,
		GuestAddress: input.GuestAddress, GuestCity: input.GuestCity, GuestCountry: input.GuestCountry, GuestPostalCode: input.GuestPostalCode,
		IdempotencyKey: input.IdempotencyKey, DiscountCode: input.DiscountCode,
		ItemTitles: input.ItemTitles, ItemSKUs: input.ItemSKUs,
	})
	if err != nil {
		if errors.Is(err, commerce.ErrOutOfStock) {
			api.WriteError(w, r, http.StatusConflict, "one or more items in your cart are no longer available in the requested quantity", nil)
			return
		}
		if errors.Is(err, commerce.ErrDiscountCodeInvalid) {
			api.WriteError(w, r, http.StatusUnprocessableEntity, "this discount code cannot be applied to your order", map[string]string{"discount_code": "invalid"})
			return
		}
		api.WriteError(w, r, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}
	response := checkoutResponse{Order: order}
	items, itemsErr := h.Service.Repository.ListOrderItems(r.Context(), h.DB, order.ID)
	if itemsErr == nil {
		if h.PaymentProvider != nil {
			paymentCtx := commerce.WithClientIP(r.Context(), clientIP(r))
			if start, startErr := h.PaymentProvider.StartPayment(paymentCtx, order, items); startErr == nil {
				response.Payment = &start
			}
		}
		h.notifyNewOrder(r.Context(), order, items)
	}
	api.WriteData(w, http.StatusCreated, response, nil)
}

// notifyNewOrder sends the customer's receipt and the operator's chat
// alert. Both best-effort: a checkout that already succeeded must never
// fail because an email or a chat ping did, so every error here is
// swallowed, never surfaced to the shopper -- the same contract
// PaymentProvider.StartPayment above already has.
func (h Handler) notifyNewOrder(ctx context.Context, order commerce.Order, items []commerce.OrderItem) {
	total := formatMoney(order.TotalCents, order.Currency)
	if h.Mailer != nil && order.GuestEmail != "" {
		mailItems := make([]mailer.OrderConfirmationItem, 0, len(items))
		for _, item := range items {
			mailItems = append(mailItems, mailer.OrderConfirmationItem{Title: item.Title, Quantity: item.Quantity, Total: formatMoney(item.TotalCents, order.Currency)})
		}
		_ = h.Mailer.SendOrderConfirmation(ctx, order.GuestEmail, order.ID, total, mailItems)
	}
	if h.Notifier != nil {
		_ = h.Notifier.Send(ctx, notify.Message{Title: "Yeni sipariş", Body: fmt.Sprintf("%s — %s", order.ID, total)})
	}
}

func formatMoney(cents int64, currency string) string {
	return fmt.Sprintf("%.2f %s", float64(cents)/100, currency)
}

type checkoutResponse struct {
	commerce.Order
	Payment *commerce.PaymentStart `json:"payment,omitempty"`
}

type orderItemView struct {
	VariantID      string `json:"variant_id"`
	SKU            string `json:"sku"`
	Title          string `json:"title"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	TotalCents     int64  `json:"total_cents"`
}

type orderView struct {
	commerce.Order
	Items []orderItemView `json:"items"`
}

// getOrder is an order confirmation/status read: a guest checkout has no
// account to log into, so knowing the order id is the only access check,
// the same trust model startOrderPayment already relies on.
func (h Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	order, err := h.Service.Repository.GetOrder(r.Context(), h.DB, r.PathValue("id"))
	if err != nil {
		api.WriteError(w, r, http.StatusNotFound, "order not found", nil)
		return
	}
	items, err := h.Service.Repository.ListOrderItems(r.Context(), h.DB, order.ID)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	views := make([]orderItemView, 0, len(items))
	for _, item := range items {
		views = append(views, orderItemView{VariantID: item.VariantID, SKU: item.SKU, Title: item.Title, Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents, TotalCents: item.TotalCents})
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, orderView{Order: order, Items: views}, nil)
}

// startOrderPayment (re)starts a payment with the configured provider for
// an existing order -- for when StartPayment at checkout time failed (a
// network blip to the provider) and the shopper needs another chance to
// pay without the cart being re-emptied. Knowing the order id is the only
// access check, the same trust model guest checkout's own confirmation
// already relies on: the id is an unguessable, app-generated token, not a
// sequential one.
func (h Handler) startOrderPayment(w http.ResponseWriter, r *http.Request) {
	if h.PaymentProvider == nil {
		api.WriteError(w, r, http.StatusServiceUnavailable, "no payment provider is configured", nil)
		return
	}
	orderID := r.PathValue("id")
	order, err := h.Service.Repository.GetOrder(r.Context(), h.DB, orderID)
	if err != nil {
		api.WriteError(w, r, http.StatusNotFound, "order not found", nil)
		return
	}
	if order.PaymentStatus != commerce.PaymentPending {
		api.WriteError(w, r, http.StatusConflict, "this order is no longer awaiting payment", nil)
		return
	}
	items, err := h.Service.Repository.ListOrderItems(r.Context(), h.DB, order.ID)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	paymentCtx := commerce.WithClientIP(r.Context(), clientIP(r))
	start, err := h.PaymentProvider.StartPayment(paymentCtx, order, items)
	if err != nil {
		api.WriteError(w, r, http.StatusBadGateway, "could not start a payment with the configured provider", nil)
		return
	}
	api.WriteData(w, http.StatusOK, start, nil)
}

// getStock is a public, unauthenticated read of how many units of a
// variant can still be added to a cart -- a storefront's product page
// needs this to show "in stock"/"sold out" without the admin-only
// reserved count, which is internal checkout bookkeeping, not a shopper
// fact. A variant nobody has stocked yet reads as zero, not an error: to a
// shopper "never stocked" and "sold out" look the same.
func (h Handler) getStock(w http.ResponseWriter, r *http.Request) {
	stock, err := h.Service.Repository.GetStock(r.Context(), h.DB, r.PathValue("variantID"))
	if err != nil {
		if errors.Is(err, commerce.ErrItemNotFound) {
			api.WriteData(w, http.StatusOK, map[string]any{"variant_id": r.PathValue("variantID"), "available": 0}, nil)
			return
		}
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"variant_id": stock.VariantID, "available": stock.Available}, nil)
}

// getCurrencies is a public, static (no DB read) config echo: a storefront
// needs this to render a currency switcher, and HydroGo's own config is
// the single source of truth for it rather than duplicating it in the
// storefront's own environment, which would drift.
func (h Handler) getCurrencies(w http.ResponseWriter, r *http.Request) {
	api.WriteData(w, http.StatusOK, map[string]any{
		"default": h.defaultCurrency(),
		"display": h.DisplayCurrencies,
		"rates":   h.CurrencyRates,
	}, nil)
}

// getCheckoutConfig is a public, static config echo (no DB read) so a
// storefront can show its tax/shipping terms, and estimate a total, before
// the shopper submits the real checkout -- the authoritative breakdown is
// always whatever the real checkout response says, this is display only.
func (h Handler) getCheckoutConfig(w http.ResponseWriter, r *http.Request) {
	pricing := h.Service.Pricing
	api.WriteData(w, http.StatusOK, map[string]any{
		"tax_rate":                  pricing.TaxRate,
		"tax_inclusive":             pricing.TaxInclusive,
		"shipping_flat_rate_cents":  pricing.ShippingFlatRateCents,
		"shipping_free_above_cents": pricing.ShippingFreeAboveCents,
	}, nil)
}

type setStockRequest struct {
	VariantID string `json:"variant_id"`
	Available int    `json:"available"`
}

func (h Handler) setStock(w http.ResponseWriter, r *http.Request) {
	var input setStockRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	if strings.TrimSpace(input.VariantID) == "" || input.Available < 0 {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "variant_id and a non-negative available are required", nil)
		return
	}
	if err := h.Service.Repository.EnsureStock(r.Context(), h.DB, input.VariantID, input.Available); err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	stock, err := h.Service.Repository.GetStock(r.Context(), h.DB, input.VariantID)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	api.WriteData(w, http.StatusOK, stock, nil)
}

type createDiscountCodeRequest struct {
	Code             string `json:"code"`
	Kind             string `json:"kind"` // "percent" or "fixed"
	Value            int64  `json:"value"`
	MinSubtotalCents int64  `json:"min_subtotal_cents"`
	MaxRedemptions   int    `json:"max_redemptions"`
	Active           *bool  `json:"active"`
	StartsAt         string `json:"starts_at"` // RFC3339; empty = no restriction
	EndsAt           string `json:"ends_at"`
}

func (h Handler) createDiscountCode(w http.ResponseWriter, r *http.Request) {
	var input createDiscountCodeRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	code := strings.TrimSpace(input.Code)
	kind := commerce.DiscountKind(input.Kind)
	if code == "" || (kind != commerce.DiscountPercent && kind != commerce.DiscountFixed) || input.Value <= 0 {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "code, a kind of percent or fixed, and a positive value are required", nil)
		return
	}
	if kind == commerce.DiscountPercent && input.Value > 100 {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "a percent discount cannot exceed 100", nil)
		return
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	var startsAt, endsAt time.Time
	var err error
	if input.StartsAt != "" {
		if startsAt, err = time.Parse(time.RFC3339, input.StartsAt); err != nil {
			api.WriteError(w, r, http.StatusUnprocessableEntity, "starts_at must be RFC3339", nil)
			return
		}
	}
	if input.EndsAt != "" {
		if endsAt, err = time.Parse(time.RFC3339, input.EndsAt); err != nil {
			api.WriteError(w, r, http.StatusUnprocessableEntity, "ends_at must be RFC3339", nil)
			return
		}
	}
	token, err := randomToken()
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	id := "disc_" + token
	record := commerce.DiscountCode{
		ID: id, Code: code, Kind: kind, Value: input.Value, MinSubtotalCents: input.MinSubtotalCents,
		MaxRedemptions: input.MaxRedemptions, Active: active, StartsAt: startsAt, EndsAt: endsAt,
	}
	if err := h.Service.Repository.CreateDiscountCode(r.Context(), h.DB, record); err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	api.WriteData(w, http.StatusCreated, record, nil)
}

// createShipment hands a paid order to the configured carrier and records
// its tracking details. Unlike checkout/payment, this is an operator
// action (or a future auto-fulfillment job), never shopper-triggered --
// hence Protected: true above, same as setStock and createDiscountCode.
func (h Handler) createShipment(w http.ResponseWriter, r *http.Request) {
	if h.ShippingProvider == nil {
		api.WriteError(w, r, http.StatusNotImplemented, "no shipping provider is configured", nil)
		return
	}
	orderID := r.PathValue("id")
	order, err := h.Service.CreateShipment(r.Context(), h.DB, h.ShippingProvider, orderID)
	if err != nil {
		switch {
		case errors.Is(err, commerce.ErrOrderNotFound):
			api.WriteError(w, r, http.StatusNotFound, "order not found", nil)
		case errors.Is(err, commerce.ErrOrderNotShippable):
			api.WriteError(w, r, http.StatusConflict, err.Error(), nil)
		default:
			api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		}
		return
	}
	api.WriteData(w, http.StatusOK, order, nil)
}
